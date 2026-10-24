package category

import (
	"api-kms/errors"
	"context"
	stderrors "errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

// GetBySpace returns every category in one space, sorted so BuildTree can nest
// them without a second sort. One flat query, not a query per level.
func (r *Repository) GetBySpace(ctx context.Context, spaceID string) ([]Category, error) {
	var categories []Category
	if err := r.db.WithContext(ctx).
		Where("space_id = ?", spaceID).
		Order("`order` ASC, name ASC").
		Find(&categories).Error; err != nil {
		return nil, errors.InternalServerError("failed to get categories").WithCause(err)
	}
	return categories, nil
}

func (r *Repository) GetById(ctx context.Context, id string) (*Category, error) {
	var category Category
	if err := r.db.WithContext(ctx).First(&category, "id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("category not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get category by id").WithCause(err)
	}
	return &category, nil
}

func (r *Repository) SlugExists(ctx context.Context, spaceID, slug, excludeID string) (bool, error) {
	db := r.db.WithContext(ctx).Model(&Category{}).Where("space_id = ? AND slug = ?", spaceID, slug)
	if excludeID != "" {
		db = db.Where("id <> ?", excludeID)
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, errors.InternalServerError("failed to check category slug").WithCause(err)
	}
	return count > 0, nil
}

// NextOrder is the position a new category takes: last among its siblings.
func (r *Repository) NextOrder(ctx context.Context, spaceID string, parentID *uuid.UUID) (int, error) {
	db := r.db.WithContext(ctx).Model(&Category{}).Where("space_id = ?", spaceID)
	db = whereParent(db, parentID)

	// COALESCE keeps this a single row rather than a nil scan on the first
	// category in a space.
	var next int
	if err := db.Select("COALESCE(MAX(`order`) + 1, 0)").Scan(&next).Error; err != nil {
		return 0, errors.InternalServerError("failed to get next category order").WithCause(err)
	}
	return next, nil
}

func (r *Repository) AddNewData(ctx context.Context, category *Category) error {
	if err := r.db.WithContext(ctx).Create(category).Error; err != nil {
		return translateDuplicateSlug(err, "failed to create category")
	}
	return nil
}

func (r *Repository) UpdateById(ctx context.Context, category *Category) error {
	if err := r.db.WithContext(ctx).Save(category).Error; err != nil {
		return translateDuplicateSlug(err, "failed to update category")
	}
	return nil
}

// DeleteById relies on the self-referencing FK's ON DELETE CASCADE for the
// subtree, and on documents.category_id's ON DELETE SET NULL for the documents
// that were filed under it — a document outlives the shelf it was on.
func (r *Repository) DeleteById(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&Category{}, "id = ?", id).Error; err != nil {
		return errors.InternalServerError("failed to delete category").WithCause(err)
	}
	return nil
}

// Reorder renumbers one sibling list 0..n-1 in a single transaction. Anything
// less leaves two categories sharing a position if the second write fails.
//
// ids that are not actually siblings of parentID are rejected by the caller
// (see Service.Reorder) — this only writes.
func (r *Repository) Reorder(ctx context.Context, ids []uuid.UUID) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for position, id := range ids {
			if err := tx.Model(&Category{}).
				Where("id = ?", id).
				Update("order", position).Error; err != nil {
				return errors.InternalServerError("failed to reorder categories").WithCause(err)
			}
		}
		return nil
	})
	return err
}

// SiblingIDs is what Reorder validates against: the ids actually filed under
// parentID in this space.
func (r *Repository) SiblingIDs(ctx context.Context, spaceID string, parentID *uuid.UUID) ([]uuid.UUID, error) {
	db := r.db.WithContext(ctx).Model(&Category{}).Where("space_id = ?", spaceID)
	db = whereParent(db, parentID)

	var ids []uuid.UUID
	if err := db.Pluck("id", &ids).Error; err != nil {
		return nil, errors.InternalServerError("failed to get sibling categories").WithCause(err)
	}
	return ids, nil
}

// whereParent keeps `IS NULL` and `= ?` in one place — GORM turns a nil
// *uuid.UUID into `= NULL`, which matches nothing.
func whereParent(db *gorm.DB, parentID *uuid.UUID) *gorm.DB {
	if parentID == nil {
		return db.Where("parent_id IS NULL")
	}
	return db.Where("parent_id = ?", *parentID)
}

func translateDuplicateSlug(err error, fallbackMessage string) error {
	if stderrors.Is(err, gorm.ErrDuplicatedKey) {
		return errors.DataAlreadyExistException("category slug already exists").
			WithErrors(map[string][]string{"name": {"a category with this name already exists in this space"}}).
			WithCause(err)
	}
	return errors.InternalServerError(fallbackMessage).WithCause(err)
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CountsBySpace returns how many live documents sit directly in each category
// of one space, plus how many sit in none of them under the uuid.Nil key.
//
// One grouped query for the whole tree, not a count per node: a three-level
// tree would otherwise cost a round trip per category every time the sidebar
// is drawn.
//
// hidden is the set of documents this caller may not see (FR-9). They are
// excluded here too — a count that includes a restricted document tells the
// people kept out of it exactly how much they are missing.
func (r *Repository) CountsBySpace(ctx context.Context, spaceID string, hidden []uuid.UUID) (map[uuid.UUID]int64, error) {
	db := r.db.WithContext(ctx).
		Table("documents").
		Select("category_id, COUNT(*) AS total").
		Where("space_id = ? AND deleted_at IS NULL", spaceID).
		Group("category_id")

	if len(hidden) > 0 {
		db = db.Where("id NOT IN ?", hidden)
	}

	var rows []struct {
		CategoryID *uuid.UUID
		Total      int64
	}
	if err := db.Scan(&rows).Error; err != nil {
		return nil, errors.InternalServerError("failed to count documents per category").WithCause(err)
	}

	counts := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		// A NULL category_id is "filed nowhere", which the caller reads back
		// under uuid.Nil rather than having to handle a second return value.
		key := uuid.Nil
		if row.CategoryID != nil {
			key = *row.CategoryID
		}
		counts[key] = row.Total
	}
	return counts, nil
}
