package tag

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

func (r *Repository) GetAll(ctx context.Context, search string) ([]Tag, error) {
	db := r.db.WithContext(ctx).Order("name ASC")
	if search != "" {
		db = db.Where("name ILIKE ?", "%"+search+"%")
	}

	var tags []Tag
	if err := db.Find(&tags).Error; err != nil {
		return nil, errors.InternalServerError("failed to get tags").WithCause(err)
	}
	return tags, nil
}

// CountsByTag returns how many documents each tag labels, in one grouped query
// rather than a count per tag.
func (r *Repository) CountsByTag(ctx context.Context) (map[string]int64, error) {
	var rows []struct {
		TagID string
		Total int64
	}
	if err := r.db.WithContext(ctx).
		Table("document_tags").
		Select("tag_id, COUNT(*) AS total").
		Group("tag_id").
		Scan(&rows).Error; err != nil {
		return nil, errors.InternalServerError("failed to count tagged documents").WithCause(err)
	}

	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.TagID] = row.Total
	}
	return counts, nil
}

func (r *Repository) GetById(ctx context.Context, id string) (*Tag, error) {
	var t Tag
	if err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("tag not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get tag by id").WithCause(err)
	}
	return &t, nil
}

// GetByDocument is what fills the tag chips on a document.
func (r *Repository) GetByDocument(ctx context.Context, documentID string) ([]Tag, error) {
	var tags []Tag
	if err := r.db.WithContext(ctx).
		Joins("JOIN document_tags ON document_tags.tag_id = tags.id").
		Where("document_tags.document_id = ?", documentID).
		Order("tags.name ASC").
		Find(&tags).Error; err != nil {
		return nil, errors.InternalServerError("failed to get document tags").WithCause(err)
	}
	return tags, nil
}

// FirstOrCreateBySlug is what makes the vocabulary open: a name nobody has used
// becomes a tag, and one that has been used resolves to the existing row
// instead of a near-duplicate.
//
// The unique index on slug is what settles a race — two requests can both miss
// the SELECT, and the loser re-reads rather than inserting a second "Deployment".
func (r *Repository) FirstOrCreateBySlug(ctx context.Context, name, slug string) (*Tag, error) {
	var t Tag
	err := r.db.WithContext(ctx).First(&t, "slug = ?", slug).Error
	if err == nil {
		return &t, nil
	}
	if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.InternalServerError("failed to look up tag").WithCause(err)
	}

	t = Tag{Name: name, Slug: slug}
	if err := r.db.WithContext(ctx).Create(&t).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			// Lost the race; the winner's row is the one everyone should use.
			if err := r.db.WithContext(ctx).First(&t, "slug = ?", slug).Error; err != nil {
				return nil, errors.InternalServerError("failed to resolve tag after conflict").WithCause(err)
			}
			return &t, nil
		}
		return nil, errors.InternalServerError("failed to create tag").WithCause(err)
	}
	return &t, nil
}

func (r *Repository) UpdateById(ctx context.Context, t *Tag) error {
	if err := r.db.WithContext(ctx).Save(t).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.DataAlreadyExistException("tag already exists").
				WithErrors(map[string][]string{"name": {"a tag with this name already exists"}}).
				WithCause(err)
		}
		return errors.InternalServerError("failed to update tag").WithCause(err)
	}
	return nil
}

// DeleteById drops the tag; document_tags goes with it via ON DELETE CASCADE,
// which is the point — deleting a tag removes the label from every document
// that carried it.
func (r *Repository) DeleteById(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&Tag{}, "id = ?", id).Error; err != nil {
		return errors.InternalServerError("failed to delete tag").WithCause(err)
	}
	return nil
}

// SyncDocumentTags replaces a document's whole set in one transaction. Delete
// then insert, rather than diffing: the set is a handful of rows, and a diff
// would be more code for the same writes.
func (r *Repository) SyncDocumentTags(ctx context.Context, documentID uuid.UUID, tagIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM document_tags WHERE document_id = ?", documentID).Error; err != nil {
			return errors.InternalServerError("failed to clear document tags").WithCause(err)
		}
		if len(tagIDs) == 0 {
			return nil
		}

		rows := make([]map[string]any, len(tagIDs))
		for i, id := range tagIDs {
			rows[i] = map[string]any{"document_id": documentID, "tag_id": id}
		}
		if err := tx.Table("document_tags").Create(rows).Error; err != nil {
			return errors.InternalServerError("failed to set document tags").WithCause(err)
		}
		return nil
	})
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetBySlug returns nil without an error when nothing holds the slug — the
// caller is asking whether it is free, and "no" is an answer rather than a
// failure.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Tag, error) {
	var t Tag
	if err := r.db.WithContext(ctx).First(&t, "slug = ?", slug).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.InternalServerError("failed to look up tag by slug").WithCause(err)
	}
	return &t, nil
}
