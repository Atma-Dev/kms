package notification

import (
	"context"
	"api-kms/errors"
	"api-kms/utils"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func (r *Repository) Create(ctx context.Context, n *Notification) error {
	if err := r.db.WithContext(ctx).Create(n).Error; err != nil {
		return errors.InternalServerError("failed to create notification").WithCause(err)
	}
	return nil
}

// GetAllPaginated lists a user's own notification feed, newest first.
func (r *Repository) GetAllPaginated(c *gin.Context, userID string) ([]Notification, *utils.PaginationMeta, error) {
	var notifications []Notification
	db := r.db.WithContext(c).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	meta, err := utils.Paginate(c, db, &notifications)
	return notifications, meta, err
}

func (r *Repository) CountUnread(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	if err != nil {
		return 0, errors.InternalServerError("failed to count unread notifications").WithCause(err)
	}
	return count, nil
}

// MarkRead scopes by user_id too, so one user can never mark another's
// notification read.
func (r *Repository) MarkRead(ctx context.Context, id string, userID string) error {
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read_at", now).Error
	if err != nil {
		return errors.InternalServerError("failed to mark notification read").WithCause(err)
	}
	return nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now).Error
	if err != nil {
		return errors.InternalServerError("failed to mark notifications read").WithCause(err)
	}
	return nil
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}
