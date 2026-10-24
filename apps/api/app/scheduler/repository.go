package scheduler

import (
	"context"
	stderrors "errors"
	"api-kms/app/joblog"
	"api-kms/errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// JobDefault is what the code knows about a job at startup — its name and
// the cron expression it ships with. Sync uses it to keep scheduled_jobs in
// step with what's actually registered; everything else about a row
// (enabled, a since-edited cron_expression) is the operator's to change.
type JobDefault struct {
	Name           string
	CronExpression string
}

// Sync inserts a row for any job the code just registered that
// scheduled_jobs doesn't know about yet, and deletes rows for jobs no
// longer registered — the DB never holds a schedule for code that isn't
// there to run it.
func (r *Repository) Sync(ctx context.Context, defaults []JobDefault) error {
	var existing []ScheduledJob
	if err := r.db.WithContext(ctx).Find(&existing).Error; err != nil {
		return errors.InternalServerError("failed to load scheduled jobs").WithCause(err)
	}
	knownNames := make(map[string]bool, len(existing))
	for _, s := range existing {
		knownNames[s.Name] = true
	}

	registeredNames := make(map[string]bool, len(defaults))
	for _, d := range defaults {
		registeredNames[d.Name] = true
		if knownNames[d.Name] {
			continue
		}
		row := ScheduledJob{Name: d.Name, CronExpression: d.CronExpression, Enabled: true}
		if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
			return errors.InternalServerError("failed to register scheduled job " + d.Name).WithCause(err)
		}
	}

	for _, s := range existing {
		if registeredNames[s.Name] {
			continue
		}
		if err := r.db.WithContext(ctx).Delete(&ScheduledJob{}, "id = ?", s.ID).Error; err != nil {
			return errors.InternalServerError("failed to remove stale scheduled job " + s.Name).WithCause(err)
		}
	}
	return nil
}

func (r *Repository) GetEnabled(ctx context.Context) ([]ScheduledJob, error) {
	var schedules []ScheduledJob
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&schedules).Error; err != nil {
		return nil, errors.InternalServerError("failed to load enabled scheduled jobs").WithCause(err)
	}
	return schedules, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]ScheduledJob, error) {
	var schedules []ScheduledJob
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&schedules).Error; err != nil {
		return nil, errors.InternalServerError("failed to load scheduled jobs").WithCause(err)
	}
	return schedules, nil
}

func (r *Repository) GetById(ctx context.Context, id string) (*ScheduledJob, error) {
	var schedule ScheduledJob
	if err := r.db.WithContext(ctx).First(&schedule, "id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("scheduled job not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get scheduled job by id").WithCause(err)
	}
	return &schedule, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*ScheduledJob, error) {
	var schedule ScheduledJob
	if err := r.db.WithContext(ctx).First(&schedule, "name = ?", name).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("scheduled job not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get scheduled job by name").WithCause(err)
	}
	return &schedule, nil
}

func (r *Repository) UpdateById(ctx context.Context, schedule *ScheduledJob) error {
	if err := r.db.WithContext(ctx).Save(schedule).Error; err != nil {
		return errors.InternalServerError("failed to update scheduled job").WithCause(err)
	}
	return nil
}

// ClaimTick is one conditional UPDATE that ties a cron tick to whichever
// instance's write actually changes last_tick_at — so when two replicas'
// tickLoop both wake up for the same minute, only one of them proceeds to
// run the job.
func (r *Repository) ClaimTick(ctx context.Context, id uuid.UUID, minute time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&ScheduledJob{}).
		Where("id = ? AND (last_tick_at IS NULL OR last_tick_at < ?)", id, minute).
		Update("last_tick_at", minute)
	if result.Error != nil {
		return false, errors.InternalServerError("failed to claim scheduled job tick").WithCause(result.Error)
	}
	return result.RowsAffected > 0, nil
}

// AcquireLease is the multi-instance (and same-instance overlap) run guard:
// a job already mid-run, with a lease that hasn't expired, refuses a second
// claimant — whether that's another instance's tick or this instance's own
// next tick landing before a slow previous run finished.
func (r *Repository) AcquireLease(ctx context.Context, id uuid.UUID, instanceID string, ttl time.Duration) (bool, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&ScheduledJob{}).
		Where("id = ? AND (locked_at IS NULL OR locked_at < ?)", id, now.Add(-ttl)).
		Updates(map[string]any{"locked_at": now, "locked_by": instanceID})
	if result.Error != nil {
		return false, errors.InternalServerError("failed to acquire scheduled job lease").WithCause(result.Error)
	}
	return result.RowsAffected > 0, nil
}

// RenewLease is the heartbeat: a job running longer than leaseTTL needs its
// lease refreshed periodically or another instance would treat it as dead
// and claim it out from under the still-running goroutine.
func (r *Repository) RenewLease(ctx context.Context, id uuid.UUID, instanceID string) error {
	result := r.db.WithContext(ctx).Model(&ScheduledJob{}).
		Where("id = ? AND locked_by = ?", id, instanceID).
		Update("locked_at", time.Now())
	if result.Error != nil {
		return errors.InternalServerError("failed to renew scheduled job lease").WithCause(result.Error)
	}
	return nil
}

func (r *Repository) ReleaseLease(ctx context.Context, id uuid.UUID, instanceID string) error {
	result := r.db.WithContext(ctx).Model(&ScheduledJob{}).
		Where("id = ? AND locked_by = ?", id, instanceID).
		Updates(map[string]any{"locked_at": nil, "locked_by": nil})
	if result.Error != nil {
		return errors.InternalServerError("failed to release scheduled job lease").WithCause(result.Error)
	}
	return nil
}

// RecordRun updates the row's denormalized last-run columns — what the
// schedules admin page shows without joining job_logs.
func (r *Repository) RecordRun(ctx context.Context, id uuid.UUID, status joblog.Status, message string, finishedAt time.Time, durationMs int64) error {
	statusStr := string(status)
	result := r.db.WithContext(ctx).Model(&ScheduledJob{}).Where("id = ?", id).Updates(map[string]any{
		"last_run_at":      finishedAt,
		"last_status":      statusStr,
		"last_message":     message,
		"last_duration_ms": durationMs,
	})
	if result.Error != nil {
		return errors.InternalServerError("failed to record scheduled job run").WithCause(result.Error)
	}
	return nil
}
