package category

import (
	"api-kms/app/document"
	"api-kms/app/user"
	"api-kms/errors"
	"api-kms/utils"
	"context"

	"github.com/google/uuid"
	goaudit "github.com/iqbalatma/go-audit"
)

type Service struct {
	repository *Repository
	// documents answers which documents this caller may not see (FR-9), so the
	// counts beside each category exclude them. A count that included a
	// restricted document would tell the people kept out of it exactly how
	// much they are missing.
	//
	// The dependency runs this way and not the other: app/document expands a
	// category filter to its subtree in SQL precisely so it needs nothing from
	// here, because two-way would not build.
	documents *document.Service
}

func (s *Service) GetTree(ctx context.Context, spaceID string) ([]Node, error) {
	rows, err := s.repository.GetBySpace(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	return BuildTree(rows), nil
}

// GetTreeWithCounts is the sidebar: the tree, each node carrying how many
// documents clicking it would list, plus the space total and how many are
// filed under no category at all.
//
// The counts deliberately ignore the status and search filters on the list
// beside them. They answer "how much is in here", which is a property of the
// tree — recomputing them on every keystroke would make the sidebar flicker
// and still not tell anyone where their documents are.
func (s *Service) GetTreeWithCounts(ctx context.Context, spaceID string, viewer *user.User) (tree []Node, total, uncategorised int64, err error) {
	rows, err := s.repository.GetBySpace(ctx, spaceID)
	if err != nil {
		return nil, 0, 0, err
	}

	hidden, err := s.documents.HiddenFrom(ctx, spaceID, viewer)
	if err != nil {
		return nil, 0, 0, err
	}

	own, err := s.repository.CountsBySpace(ctx, spaceID, hidden)
	if err != nil {
		return nil, 0, 0, err
	}

	tree = BuildTree(rows)
	inCategories := Rollup(tree, own)
	uncategorised = own[uuid.Nil]

	return tree, inCategories + uncategorised, uncategorised, nil
}

func (s *Service) AddNew(ctx context.Context, spaceID string, request *StoreCategoryRequest) (*Category, error) {
	if err := s.checkParent(ctx, spaceID, request.ParentID, ""); err != nil {
		return nil, err
	}

	slug, err := s.uniqueSlug(ctx, spaceID, request.Name, "")
	if err != nil {
		return nil, err
	}

	order, err := s.repository.NextOrder(ctx, spaceID, request.ParentID)
	if err != nil {
		return nil, err
	}

	space, err := uuid.Parse(spaceID)
	if err != nil {
		return nil, errors.EmptyDataException("space not found").WithCause(err)
	}

	category := Category{
		SpaceID:  space,
		ParentID: request.ParentID,
		Name:     request.Name,
		Slug:     slug,
		Order:    order,
	}
	if err := s.repository.AddNewData(ctx, &category); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "CREATE_CATEGORY", "create category via category.Service.AddNew").
		SetEntryObject("categories", category.ID.String()).
		AddSingleTrail("categories", category.ID.String(), nil, map[string]any{
			"space_id": spaceID, "name": category.Name, "slug": category.Slug, "parent_id": category.ParentID,
		}, nil, nil).
		Execute(ctx)

	return &category, nil
}

func (s *Service) UpdateById(ctx context.Context, id string, request *UpdateCategoryRequest) (*Category, error) {
	category, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	spaceID := category.SpaceID.String()
	if err := s.checkParent(ctx, spaceID, request.ParentID, id); err != nil {
		return nil, err
	}

	before := map[string]any{"name": category.Name, "slug": category.Slug, "parent_id": category.ParentID}

	// Same rule as app/space: the slug moves only when the name does.
	if request.Name != category.Name {
		slug, err := s.uniqueSlug(ctx, spaceID, request.Name, id)
		if err != nil {
			return nil, err
		}
		category.Slug = slug
	}

	// Moving to a new parent puts it last there — its old position means
	// nothing among its new siblings.
	if !sameParent(category.ParentID, request.ParentID) {
		order, err := s.repository.NextOrder(ctx, spaceID, request.ParentID)
		if err != nil {
			return nil, err
		}
		category.Order = order
		category.ParentID = request.ParentID
	}

	category.Name = request.Name
	if err := s.repository.UpdateById(ctx, category); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "UPDATE_CATEGORY", "update category via category.Service.UpdateById").
		SetEntryObject("categories", category.ID.String()).
		AddSingleTrail("categories", category.ID.String(), before, map[string]any{
			"name": category.Name, "slug": category.Slug, "parent_id": category.ParentID,
		}, nil, nil).
		Execute(ctx)

	return category, nil
}

func (s *Service) DeleteById(ctx context.Context, id string) error {
	category, err := s.repository.GetById(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.DeleteById(ctx, id); err != nil {
		return err
	}

	goaudit.Init(ctx, "DELETE_CATEGORY", "delete category via category.Service.DeleteById").
		SetEntryObject("categories", id).
		AddSingleTrail("categories", id,
			map[string]any{"name": category.Name, "slug": category.Slug}, nil, nil, nil).
		Execute(ctx)

	return nil
}

// Reorder renumbers one sibling list. The submitted ids must be exactly that
// list — a partial list would renumber some siblings and leave the rest
// pointing at positions that no longer mean anything.
func (s *Service) Reorder(ctx context.Context, spaceID string, request *ReorderCategoriesRequest) error {
	siblings, err := s.repository.SiblingIDs(ctx, spaceID, request.ParentID)
	if err != nil {
		return err
	}

	if len(siblings) != len(request.IDs) {
		return errors.InvalidStatusException("the ordered list must contain every category at this level")
	}

	known := make(map[uuid.UUID]bool, len(siblings))
	for _, id := range siblings {
		known[id] = true
	}
	for _, id := range request.IDs {
		if !known[id] {
			return errors.InvalidStatusException("category " + id.String() + " is not at this level")
		}
		// Guards a duplicate id padding the list to the right length.
		delete(known, id)
	}

	if err := s.repository.Reorder(ctx, request.IDs); err != nil {
		return err
	}

	goaudit.Init(ctx, "REORDER_CATEGORIES", "reorder categories via category.Service.Reorder").
		SetEntryObject("categories", spaceID).
		AddSingleTrail("categories", spaceID, nil, map[string]any{"ids": request.IDs}, nil, nil).
		Execute(ctx)

	return nil
}

// checkParent enforces FR-2's three levels and refuses a move that would put a
// category inside its own subtree.
func (s *Service) checkParent(ctx context.Context, spaceID string, parentID *uuid.UUID, movingID string) error {
	if parentID == nil {
		return nil
	}

	parent, err := s.repository.GetById(ctx, parentID.String())
	if err != nil {
		return err
	}
	if parent.SpaceID.String() != spaceID {
		return errors.InvalidStatusException("the parent category belongs to another space")
	}
	if movingID != "" && parent.ID.String() == movingID {
		return errors.InvalidStatusException("a category cannot be its own parent")
	}

	// Walk to the root, counting. The cap is 3, so this is at most two hops —
	// and the loop doubles as the cycle check for the move case.
	depth := 1
	for current := parent; current.ParentID != nil; {
		if movingID != "" && current.ParentID.String() == movingID {
			return errors.InvalidStatusException("a category cannot be moved inside itself")
		}
		depth++
		if depth >= MaxDepth {
			return errors.InvalidStatusException("categories may not nest more than 3 levels deep")
		}
		if current, err = s.repository.GetById(ctx, current.ParentID.String()); err != nil {
			return err
		}
	}

	return nil
}

func sameParent(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (s *Service) uniqueSlug(ctx context.Context, spaceID, name, excludeID string) (string, error) {
	return utils.UniqueSlug(utils.Slugify(name), func(candidate string) (bool, error) {
		return s.repository.SlugExists(ctx, spaceID, candidate, excludeID)
	})
}

func NewService(repository *Repository, documents *document.Service) *Service {
	return &Service{repository: repository, documents: documents}
}
