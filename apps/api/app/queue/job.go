// Package queue is a durable, event-driven job queue — the counterpart to
// app/scheduler's periodic jobs. queue_jobs is the source of truth for a
// job's current state; every attempt is additionally written to job_logs
// with type="queue" (via joblog.Broadcaster), so the run-history page shows
// both scheduled and queue runs from one table.
//
// Postgres holds the jobs; Redis (queue.Notifier) is only a doorbell that wakes
// an idle worker early. Workers claim rows with SELECT ... FOR UPDATE SKIP
// LOCKED and also poll on a fixed interval, so a lost Redis message costs
// latency, never a job.
//
// Adding a queue job: implement queue.JobHandler and register it in
// routes.NewContainer's queue.NewRegistry(...) call. That is all.
package queue

import (
	"api-kms/app"
	"time"
)

const (
	DefaultQueue       = "default"
	DefaultMaxAttempts = 3
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusSuccess  Status = "success"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
)

func (s Status) IsFinished() bool {
	return s == StatusSuccess || s == StatusFailed || s == StatusCanceled
}

// Job is one durable unit of queued work.
type Job struct {
	app.BaseModel `gorm:"embedded"`
	Queue         string     `json:"queue" gorm:"column:queue"`
	Name          string     `json:"name" gorm:"column:name"`
	Payload       []byte     `json:"payload" gorm:"column:payload;type:json"`
	Status        Status     `json:"status" gorm:"column:status"`
	Attempts      int        `json:"attempts" gorm:"column:attempts"`
	MaxAttempts   int        `json:"max_attempts" gorm:"column:max_attempts"`
	AvailableAt   time.Time  `json:"available_at" gorm:"column:available_at"`
	ReservedAt    *time.Time `json:"reserved_at" gorm:"column:reserved_at"`
	StartedAt     *time.Time `json:"started_at" gorm:"column:started_at"`
	FinishedAt    *time.Time `json:"finished_at" gorm:"column:finished_at"`
	LastError     *string    `json:"last_error" gorm:"column:last_error"`
	DispatchedBy  *string    `json:"dispatched_by" gorm:"column:dispatched_by"`
}

func (Job) TableName() string {
	return "queue_jobs"
}

// Backoff computes an exponential retry delay for attempt n (1-indexed),
// capped at max, so a job that keeps failing backs off instead of hammering
// whatever it depends on.
func Backoff(attempt int, base, max time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := base
	for i := 1; i < attempt; i++ {
		if delay >= max {
			return max
		}
		delay *= 2
	}
	if delay > max {
		return max
	}
	return delay
}
