package queue

import (
	"context"
	stderrors "errors"
	"api-kms/errors"
	"api-kms/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, job *Job) error {
	if err := r.db.WithContext(ctx).Create(job).Error; err != nil {
		return errors.InternalServerError("failed to create queue job").WithCause(err)
	}
	return nil
}

// CreateTx inserts inside a caller-managed transaction — see
// Dispatcher.Transaction, the only sanctioned way to dispatch a job that
// must commit or vanish with the caller's own writes.
func (r *Repository) CreateTx(ctx context.Context, tx *gorm.DB, job *Job) error {
	if err := tx.WithContext(ctx).Create(job).Error; err != nil {
		return errors.InternalServerError("failed to create queue job").WithCause(err)
	}
	return nil
}

func (r *Repository) GetById(ctx context.Context, id string) (*Job, error) {
	var job Job
	if err := r.db.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("queue job not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get queue job by id").WithCause(err)
	}
	return &job, nil
}

func (r *Repository) GetAllPaginated(c *gin.Context) ([]Job, *utils.PaginationMeta, error) {
	db := r.db.WithContext(c.Request.Context()).Order("created_at DESC")
	if queueName := c.Query("queue"); queueName != "" {
		db = db.Where("queue = ?", queueName)
	}
	if name := c.Query("name"); name != "" {
		db = db.Where("name = ?", name)
	}
	if status := c.Query("status"); status != "" {
		db = db.Where("status = ?", status)
	}

	var jobs []Job
	meta, err := utils.Paginate(c, db, &jobs)
	return jobs, meta, err
}

// Claim atomically reserves the oldest available pending job on one of the
// given queues. SELECT ... FOR UPDATE SKIP LOCKED means concurrent workers
// — this instance's own pool, or another instance's — never claim the same
// row; the attempt counter moves here, at claim time, so a worker that dies
// mid-run still spends the attempt rather than getting it back for free.
//
// Uses Find into a slice rather than Take/First: an empty result is the
// normal "nothing to do this poll" case, and First would turn that into a
// gorm.ErrRecordNotFound logged at error level every few seconds.
func (r *Repository) Claim(ctx context.Context, queues []string) (*Job, error) {
	var claimed *Job
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var found []Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", StatusPending).
			Where("queue IN ?", queues).
			Where("available_at <= ?", time.Now()).
			Order("available_at ASC").
			Limit(1).
			Find(&found).Error; err != nil {
			return err
		}
		if len(found) == 0 {
			return nil
		}

		job := found[0]
		now := time.Now()
		job.Status = StatusRunning
		job.Attempts++
		job.ReservedAt = &now
		job.StartedAt = &now
		job.FinishedAt = nil

		if err := tx.Model(&Job{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status": job.Status, "attempts": job.Attempts,
			"reserved_at": now, "started_at": now, "finished_at": nil,
		}).Error; err != nil {
			return err
		}
		claimed = &job
		return nil
	})
	if err != nil {
		return nil, errors.InternalServerError("failed to claim queue job").WithCause(err)
	}
	return claimed, nil
}

func (r *Repository) MarkSucceeded(ctx context.Context, job *Job, finishedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", job.ID).Updates(map[string]any{
		"status": StatusSuccess, "finished_at": finishedAt, "reserved_at": nil, "last_error": nil,
	})
	if result.Error != nil {
		return errors.InternalServerError("failed to mark queue job succeeded").WithCause(result.Error)
	}
	return nil
}

// MarkFailed either schedules a retry (retryAt non-nil: back to pending, at
// that time) or fails the job outright (retryAt nil: attempts exhausted or
// the handler returned a Permanent error).
func (r *Repository) MarkFailed(ctx context.Context, job *Job, message string, finishedAt time.Time, retryAt *time.Time) error {
	updates := map[string]any{"last_error": message, "reserved_at": nil}
	if retryAt != nil {
		updates["status"] = StatusPending
		updates["available_at"] = *retryAt
		updates["finished_at"] = nil
	} else {
		updates["status"] = StatusFailed
		updates["finished_at"] = finishedAt
	}
	result := r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", job.ID).Updates(updates)
	if result.Error != nil {
		return errors.InternalServerError("failed to mark queue job failed").WithCause(result.Error)
	}
	return nil
}

// ClearPayload blanks a finished sensitive job's payload — see
// Registry.SensitivePayload.
func (r *Repository) ClearPayload(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", id).Update("payload", nil)
	if result.Error != nil {
		return errors.InternalServerError("failed to clear queue job payload").WithCause(result.Error)
	}
	return nil
}

// Retry resets a finished job back to pending, as if freshly dispatched.
// Refused on a job still pending/running — retrying work that hasn't
// finished doesn't mean anything.
func (r *Repository) Retry(ctx context.Context, id string) (*Job, error) {
	job, err := r.GetById(ctx, id)
	if err != nil {
		return nil, err
	}
	if !job.Status.IsFinished() {
		return nil, errors.InvalidAction("only a finished queue job can be retried")
	}

	result := r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", id).Updates(map[string]any{
		"status": StatusPending, "attempts": 0, "available_at": time.Now(),
		"started_at": nil, "finished_at": nil, "reserved_at": nil, "last_error": nil,
	})
	if result.Error != nil {
		return nil, errors.InternalServerError("failed to retry queue job").WithCause(result.Error)
	}
	return r.GetById(ctx, id)
}

// RetryFailed bulk-resets every failed job (optionally narrowed to one
// name) back to pending. WHERE-guarded on status='failed' so a job a
// worker just picked up isn't reset out from under it.
func (r *Repository) RetryFailed(ctx context.Context, name string) (int64, error) {
	db := r.db.WithContext(ctx).Model(&Job{}).Where("status = ?", StatusFailed)
	if name != "" {
		db = db.Where("name = ?", name)
	}
	result := db.Updates(map[string]any{
		"status": StatusPending, "attempts": 0, "available_at": time.Now(),
		"started_at": nil, "finished_at": nil, "reserved_at": nil, "last_error": nil,
	})
	if result.Error != nil {
		return 0, errors.InternalServerError("failed to retry failed queue jobs").WithCause(result.Error)
	}
	return result.RowsAffected, nil
}

// Cancel refuses a job already claimed by a worker — canceling only means
// something before a job starts running.
func (r *Repository) Cancel(ctx context.Context, id string) (*Job, error) {
	result := r.db.WithContext(ctx).Model(&Job{}).
		Where("id = ? AND status = ?", id, StatusPending).
		Updates(map[string]any{"status": StatusCanceled, "finished_at": time.Now()})
	if result.Error != nil {
		return nil, errors.InternalServerError("failed to cancel queue job").WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, errors.InvalidAction("only a pending queue job can be canceled")
	}
	return r.GetById(ctx, id)
}

// ReleaseStale recovers jobs a worker died mid-run holding: past-budget
// attempts fail outright, everything else goes back to pending.
func (r *Repository) ReleaseStale(ctx context.Context, olderThan time.Duration) (released int64, failedCount int64, err error) {
	cutoff := time.Now().Add(-olderThan)
	const message = "worker died before the job reported back"

	failResult := r.db.WithContext(ctx).Model(&Job{}).
		Where("status = ? AND reserved_at < ? AND attempts >= max_attempts", StatusRunning, cutoff).
		Updates(map[string]any{"status": StatusFailed, "finished_at": time.Now(), "reserved_at": nil, "last_error": message})
	if failResult.Error != nil {
		return 0, 0, errors.InternalServerError("failed to fail stale queue jobs").WithCause(failResult.Error)
	}

	releaseResult := r.db.WithContext(ctx).Model(&Job{}).
		Where("status = ? AND reserved_at < ?", StatusRunning, cutoff).
		Updates(map[string]any{"status": StatusPending, "reserved_at": nil, "started_at": nil, "last_error": message})
	if releaseResult.Error != nil {
		return 0, failResult.RowsAffected, errors.InternalServerError("failed to release stale queue jobs").WithCause(releaseResult.Error)
	}

	return releaseResult.RowsAffected, failResult.RowsAffected, nil
}

// Prune deletes only finished, non-failed rows older than cutoff — a failed
// job is evidence someone may still need to look at, so it's never
// auto-pruned.
func (r *Repository) Prune(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status IN ?", []Status{StatusSuccess, StatusCanceled}).
		Where("finished_at < ?", cutoff).
		Delete(&Job{})
	if result.Error != nil {
		return 0, errors.InternalServerError("failed to prune queue jobs").WithCause(result.Error)
	}
	return result.RowsAffected, nil
}

// Purge is the manual "clear finished" button — any subset of finished
// statuses the caller asks for (PurgeRequest's binding excludes
// pending/running).
func (r *Repository) Purge(ctx context.Context, statuses []Status) (int64, error) {
	result := r.db.WithContext(ctx).Where("status IN ?", statuses).Delete(&Job{})
	if result.Error != nil {
		return 0, errors.InternalServerError("failed to purge queue jobs").WithCause(result.Error)
	}
	return result.RowsAffected, nil
}

func (r *Repository) DistinctQueues(ctx context.Context) ([]string, error) {
	var queues []string
	if err := r.db.WithContext(ctx).Model(&Job{}).Distinct().Pluck("queue", &queues).Error; err != nil {
		return nil, errors.InternalServerError("failed to list queue names").WithCause(err)
	}
	return queues, nil
}
