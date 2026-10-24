package protection

import (
	"api-kms/errors"
	"api-kms/utils"
	"context"
	stderrors "errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// RecordAccess appends one row. There is no update and no delete beside it:
// the table is append-only (FR-4), and a repository that cannot express a
// change is the cheapest way to keep it that way.
func (r *Repository) RecordAccess(ctx context.Context, log *AccessLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return errors.InternalServerError("failed to record document access").WithCause(err)
	}
	return nil
}

// AccessLogFilter is the shape both listings take. Exactly one of DocumentID
// and UserID is set by the handler — the two routes are "everything that
// happened to this document" and "everything this person did".
type AccessLogFilter struct {
	DocumentID string
	UserID     string
	Action     Action
	From       *time.Time
	To         *time.Time
}

func (r *Repository) AccessLogsPaginated(c *gin.Context, filter AccessLogFilter) ([]AccessLog, *utils.PaginationMeta, error) {
	db := r.db.WithContext(c.Request.Context()).
		Model(&AccessLog{}).
		Preload("Document").
		Preload("User").
		Order("created_at DESC")

	if filter.DocumentID != "" {
		db = db.Where("document_id = ?", filter.DocumentID)
	}
	if filter.UserID != "" {
		db = db.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		db = db.Where("action = ?", filter.Action)
	}
	if filter.From != nil {
		db = db.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		db = db.Where("created_at < ?", *filter.To)
	}

	var logs []AccessLog
	meta, err := utils.Paginate(c, db, &logs)
	return logs, meta, err
}

func (r *Repository) CreateSession(ctx context.Context, session *ViewSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return errors.InternalServerError("failed to create view session").WithCause(err)
	}
	return nil
}

// SessionsFor lists the live links to one document. Expired rows are left out
// rather than shown struck through: the list exists to answer "who can open
// this right now", and a row that no longer can is noise in that answer.
func (r *Repository) SessionsFor(ctx context.Context, documentID string) ([]ViewSession, error) {
	var sessions []ViewSession
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Where("document_id = ? AND expires_at > ?", documentID, time.Now()).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, errors.InternalServerError("failed to get view sessions").WithCause(err)
	}
	return sessions, nil
}

// SessionByTokenHash resolves a share link. An expired row is returned rather
// than hidden so the caller can say "this link has expired", which is a
// different thing for the recipient to act on than "no such link".
func (r *Repository) SessionByTokenHash(ctx context.Context, hash string) (*ViewSession, error) {
	// The creator is preloaded because the recipient's watermark names them:
	// they have no account of their own, so the trail runs to whoever shared it.
	var session ViewSession
	if err := r.db.WithContext(ctx).Preload("Creator").First(&session, "token_hash = ?", hash).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("view link not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get view session").WithCause(err)
	}
	return &session, nil
}

// DeleteSession revokes a link early, scoped to the document it belongs to so
// an id from one document cannot revoke a link on another.
func (r *Repository) DeleteSession(ctx context.Context, documentID, sessionID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND document_id = ?", sessionID, documentID).
		Delete(&ViewSession{})
	if result.Error != nil {
		return errors.InternalServerError("failed to revoke view session").WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.EmptyDataException("view link not found")
	}
	return nil
}

// DeleteExpiredSessions is the scheduler's sweep. A row past its expiry grants
// nothing, so it is only a row that makes the unique index longer and the
// listing query slower.
func (r *Repository) DeleteExpiredSessions(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("expires_at < ?", before).Delete(&ViewSession{})
	if result.Error != nil {
		return 0, errors.InternalServerError("failed to prune view sessions").WithCause(result.Error)
	}
	return result.RowsAffected, nil
}
