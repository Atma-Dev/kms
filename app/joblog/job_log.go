package joblog

import (
	"api-kms/app"
	"time"
)

// Type distinguishes what triggered the run: a schedule (time-based, the
// only source today) or a queue (event-based, once one exists). Both are
// "a unit of background work ran" and share every other column, so one
// table serves both instead of a duplicate log per source.
type Type string

const (
	TypeSchedule Type = "schedule"
	TypeQueue    Type = "queue"
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	// StatusSkipped marks a scheduled tick that found the job already running
	// on another instance (or this one) — see scheduler.Runner's lease. It is
	// not a failure: the job simply didn't need to run twice.
	StatusSkipped Status = "skipped"
)

// JobLog is one run of a background job — written after the run finishes
// (success or failure), not while it's in flight.
type JobLog struct {
	app.BaseModel `gorm:"embedded"`
	Type          Type       `json:"type" gorm:"column:type"`
	Name          string     `json:"name" gorm:"column:name"`
	Status        Status     `json:"status" gorm:"column:status"`
	Message       *string    `json:"message" gorm:"column:message"`
	StartedAt     time.Time  `json:"started_at" gorm:"column:started_at"`
	FinishedAt    *time.Time `json:"finished_at" gorm:"column:finished_at"`
}

func (JobLog) TableName() string {
	return "job_logs"
}
