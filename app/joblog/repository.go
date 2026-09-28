package joblog

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

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, log *JobLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return errors.InternalServerError("failed to create job log").WithCause(err)
	}
	return nil
}

func (r *Repository) GetAllPaginated(c *gin.Context) ([]JobLog, *utils.PaginationMeta, error) {
	db := r.db.WithContext(c).Order("started_at DESC")

	if jobType := c.Query("type"); jobType != "" {
		db = db.Where("type = ?", jobType)
	}
	if name := c.Query("name"); name != "" {
		db = db.Where("name = ?", name)
	}
	if status := c.Query("status"); status != "" {
		db = db.Where("status = ?", status)
	}

	var logs []JobLog
	meta, err := utils.Paginate(c, db, &logs)
	return logs, meta, err
}

// Prune deletes every run older than cutoff — app/scheduler's own
// prune-job-logs job, keeping this table from growing forever (two jobs a
// minute is ~2,880 rows/day before this exists).
func (r *Repository) Prune(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("started_at < ?", cutoff).Delete(&JobLog{})
	if result.Error != nil {
		return 0, errors.InternalServerError("failed to prune job logs").WithCause(result.Error)
	}
	return result.RowsAffected, nil
}
