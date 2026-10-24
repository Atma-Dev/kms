package attachment

import (
	"api-kms/errors"
	"context"
	stderrors "errors"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

// GetByDocument lists the downloadable attachments only. Inline images share
// the table but are already visible in the content — listing them again would
// be the same picture twice.
func (r *Repository) GetByDocument(ctx context.Context, documentID string) ([]Attachment, error) {
	var attachments []Attachment
	if err := r.db.WithContext(ctx).
		Where("document_id = ? AND kind = ?", documentID, KindAttachment).
		Order("created_at DESC").
		Find(&attachments).Error; err != nil {
		return nil, errors.InternalServerError("failed to get attachments").WithCause(err)
	}
	return attachments, nil
}

func (r *Repository) GetById(ctx context.Context, id string) (*Attachment, error) {
	var a Attachment
	if err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("attachment not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get attachment by id").WithCause(err)
	}
	return &a, nil
}

func (r *Repository) AddNewData(ctx context.Context, a *Attachment) error {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return errors.InternalServerError("failed to record attachment").WithCause(err)
	}
	return nil
}

func (r *Repository) DeleteById(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&Attachment{}, "id = ?", id).Error; err != nil {
		return errors.InternalServerError("failed to delete attachment").WithCause(err)
	}
	return nil
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// OrphanedInline lists inline images older than cutoff that never made it into
// a document.
//
// Rows come back whole rather than as a bulk DELETE, because each one owns a
// file: deleting the rows in one statement would strand every file they name,
// which is the problem this job exists to solve.
func (r *Repository) OrphanedInline(ctx context.Context, cutoff time.Time) ([]Attachment, error) {
	var orphans []Attachment
	if err := r.db.WithContext(ctx).
		Where("kind = ? AND document_id IS NULL AND created_at < ?", KindInline, cutoff).
		Find(&orphans).Error; err != nil {
		return nil, errors.InternalServerError("failed to list orphaned uploads").WithCause(err)
	}
	return orphans, nil
}
