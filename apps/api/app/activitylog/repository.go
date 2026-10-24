package activitylog

import (
	"context"
	"api-kms/errors"
	"api-kms/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

// Create only ever writes the activity_logs row. middleware.LoggerMiddleware
// sets log.User (the full, already-loaded row from context) so a broadcast
// payload can be built without another query — Omit(clause.Associations)
// keeps GORM from treating that as "save the user too", which would re-save
// every column on users on the way past.
func (r *Repository) Create(ctx context.Context, log *ActivityLog) error {
	if err := r.db.WithContext(ctx).Omit(clause.Associations).Create(log).Error; err != nil {
		return errors.InternalServerError("failed to create activity log").WithCause(err)
	}
	return nil
}

func (r *Repository) GetAllPaginated(c *gin.Context) ([]ActivityLog, *utils.PaginationMeta, error) {
	var logs []ActivityLog
	db := r.db.WithContext(c).
		Preload("User").
		Order("created_at DESC")

	if search := c.Query("search"); search != "" {
		db = db.Where("path ILIKE ?", "%"+search+"%")
	}
	if userID := c.Query("user_id"); userID != "" {
		db = db.Where("user_id = ?", userID)
	}
	if method := c.Query("method"); method != "" {
		db = db.Where("method = ?", strings.ToUpper(method))
	}
	// Malformed status_code is treated as "no filter" rather than a 400 —
	// this is a filter input, not a bound request field.
	if statusCode := c.Query("status_code"); statusCode != "" {
		if parsed, err := strconv.Atoi(statusCode); err == nil {
			db = db.Where("status_code = ?", parsed)
		}
	}
	// created_at compared directly (not wrapped in DATE()) so the range can
	// still use an index; adding a day gives date_to the same "whole day
	// inclusive" behavior as the deposits list filter.
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		db = db.Where("created_at >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		db = db.Where("created_at < ?::date + INTERVAL '1 day'", dateTo)
	}

	meta, err := utils.Paginate(c, db, &logs)
	return logs, meta, err
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}
