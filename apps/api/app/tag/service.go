package tag

import (
	"api-kms/errors"
	"api-kms/utils"
	"context"
	"strings"

	"github.com/google/uuid"
	goaudit "github.com/iqbalatma/go-audit"
)

type Service struct {
	repository *Repository
}

func (s *Service) GetAllWithCounts(ctx context.Context, search string) ([]Tag, map[string]int64, error) {
	tags, err := s.repository.GetAll(ctx, search)
	if err != nil {
		return nil, nil, err
	}
	counts, err := s.repository.CountsByTag(ctx)
	if err != nil {
		return nil, nil, err
	}
	return tags, counts, nil
}

func (s *Service) GetByDocument(ctx context.Context, documentID string) ([]Tag, error) {
	return s.repository.GetByDocument(ctx, documentID)
}

// AddNew is the tag master's create. Tagging a document creates tags on its
// own (see SyncDocumentTags) — this exists for seeding the vocabulary ahead of
// time, and answers with the existing tag rather than a 409 when the name is
// already taken: the caller wanted that tag to exist, and it does.
func (s *Service) AddNew(ctx context.Context, request *StoreTagRequest) (*Tag, error) {
	name, slug, err := normalize(request.Name)
	if err != nil {
		return nil, err
	}

	t, err := s.repository.FirstOrCreateBySlug(ctx, name, slug)
	if err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "CREATE_TAG", "create tag via tag.Service.AddNew").
		SetEntryObject("tags", t.ID.String()).
		AddSingleTrail("tags", t.ID.String(), nil, map[string]any{"name": t.Name, "slug": t.Slug}, nil, nil).
		Execute(ctx)

	return t, nil
}

// DeleteById removes the tag from every document that carried it — the
// cascade on document_tags is the point, not a side effect.
func (s *Service) DeleteById(ctx context.Context, id string) error {
	t, err := s.repository.GetById(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.DeleteById(ctx, id); err != nil {
		return err
	}

	goaudit.Init(ctx, "DELETE_TAG", "delete tag via tag.Service.DeleteById").
		SetEntryObject("tags", id).
		AddSingleTrail("tags", id, map[string]any{"name": t.Name, "slug": t.Slug}, nil, nil, nil).
		Execute(ctx)

	return nil
}

// SyncDocument replaces a document's tag set, creating any name that is new.
// Returns the tags the document now carries, so the caller renders what was
// stored rather than what it sent — the two differ whenever a name resolved to
// an existing tag with different casing.
func (s *Service) SyncDocument(ctx context.Context, documentID uuid.UUID, request *SyncDocumentTagsRequest) ([]Tag, error) {
	seen := make(map[string]bool, len(request.Names))
	ids := make([]uuid.UUID, 0, len(request.Names))

	for _, raw := range request.Names {
		name, slug, err := normalize(raw)
		if err != nil {
			return nil, err
		}
		// "Deploy" and "deploy" in the same submission are one tag, and the
		// composite primary key would reject the second row anyway.
		if seen[slug] {
			continue
		}
		seen[slug] = true

		t, err := s.repository.FirstOrCreateBySlug(ctx, name, slug)
		if err != nil {
			return nil, err
		}
		ids = append(ids, t.ID)
	}

	if err := s.repository.SyncDocumentTags(ctx, documentID, ids); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "SYNC_DOCUMENT_TAGS", "set document tags via tag.Service.SyncDocument").
		SetEntryObject("documents", documentID.String()).
		AddSingleTrail("document_tags", documentID.String(), nil, map[string]any{"names": request.Names}, nil, nil).
		Execute(ctx)

	return s.repository.GetByDocument(ctx, documentID.String())
}

// normalize trims the name and derives the slug that decides identity. A name
// that slugs to nothing (punctuation only) is refused rather than stored as
// "untitled" — unlike a document, a tag nobody can read is not worth keeping.
func normalize(raw string) (name, slug string, err error) {
	name = strings.TrimSpace(raw)
	slug = utils.Slugify(name)
	if slug == "" {
		return "", "", errors.InvalidStatusException("a tag needs at least one letter or digit").
			WithErrors(map[string][]string{"names": {"a tag needs at least one letter or digit"}})
	}
	return name, slug, nil
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

// UpdateById renames a tag, which relabels every document already carrying it.
//
// A rename onto a name that already exists is refused rather than merged. The
// two are different acts — merging silently folds one vocabulary entry into
// another across every document using it, and there is no undo — so the caller
// is told to delete the redundant tag instead, which at least says what it
// will cost.
func (s *Service) UpdateById(ctx context.Context, id string, request *UpdateTagRequest) (*Tag, error) {
	t, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	name, slug, err := normalize(request.Name)
	if err != nil {
		return nil, err
	}

	before := map[string]any{"name": t.Name, "slug": t.Slug}

	if slug != t.Slug {
		existing, err := s.repository.GetBySlug(ctx, slug)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, errors.DataAlreadyExistException("a tag named \"" + existing.Name + "\" already exists — delete one of them instead of renaming onto it").
				WithErrors(map[string][]string{"name": {"this tag already exists"}})
		}
	}

	t.Name = name
	t.Slug = slug
	if err := s.repository.UpdateById(ctx, t); err != nil {
		return nil, err
	}

	goaudit.Init(ctx, "UPDATE_TAG", "rename tag via tag.Service.UpdateById").
		SetEntryObject("tags", t.ID.String()).
		AddSingleTrail("tags", t.ID.String(), before, map[string]any{"name": t.Name, "slug": t.Slug}, nil, nil).
		Execute(ctx)

	return t, nil
}
