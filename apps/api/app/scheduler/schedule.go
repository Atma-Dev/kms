package scheduler

import (
	"api-kms/app"
	"time"
)

// ScheduledJob is the DB-editable half of a Job: the code decides what jobs
// exist (Job.Name/DefaultCron), this row decides when one actually runs,
// whether it's enabled, and — via LockedAt/LockedBy — which instance is
// currently running it, so two replicas never run the same tick twice.
type ScheduledJob struct {
	app.BaseModel  `gorm:"embedded"`
	Name           string     `json:"name" gorm:"column:name"`
	CronExpression string     `json:"cron_expression" gorm:"column:cron_expression"`
	Enabled        bool       `json:"enabled" gorm:"column:enabled"`
	LastTickAt     *time.Time `json:"last_tick_at" gorm:"column:last_tick_at"`
	LockedAt       *time.Time `json:"locked_at" gorm:"column:locked_at"`
	LockedBy       *string    `json:"locked_by" gorm:"column:locked_by"`
	LastRunAt      *time.Time `json:"last_run_at" gorm:"column:last_run_at"`
	LastStatus     *string    `json:"last_status" gorm:"column:last_status"`
	LastMessage    *string    `json:"last_message" gorm:"column:last_message"`
	LastDurationMs *int64     `json:"last_duration_ms" gorm:"column:last_duration_ms"`
}

func (ScheduledJob) TableName() string {
	return "scheduled_jobs"
}
