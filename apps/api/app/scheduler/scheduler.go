// Package scheduler runs this app's own periodic background jobs.
//
// A job is Go code (the Job interface) registered once at startup; its
// *schedule* is a row in scheduled_jobs that an operator can edit, disable
// or trigger by hand from the management UI. The code decides what exists,
// the table decides when.
//
// Runner ticks once a minute, on the minute, and for every enabled schedule
// whose cron expression matches, claims the tick (ClaimTick) and the run
// itself (AcquireLease) before calling Job.Run — the two guards a single
// ticker+interval design didn't need but a fleet of replicas does: ClaimTick
// stops two instances from both reacting to the same minute, AcquireLease
// stops a still-running previous invocation from being started again by the
// next tick.
package scheduler

import (
	"context"
	"fmt"
	"api-kms/app/joblog"
	"api-kms/config"
	"time"

	"github.com/google/uuid"
)

// Job is one background task the scheduler knows how to run. DefaultCron is
// only the value scheduled_jobs is seeded with the first time this job is
// seen — once a row exists, its cron_expression is the operator's to edit,
// and this method is never consulted again.
type Job interface {
	Name() string
	DefaultCron() string
	Run(ctx context.Context) (message string, err error)
}

// Runner drives every registered Job against its DB-backed schedule and
// records each run via joblog.Broadcaster.
type Runner struct {
	repository *Repository
	joblogs    *joblog.Broadcaster
	jobs       map[string]Job
	instanceID string
	leaseTTL   time.Duration
	trigger    chan string
}

func NewRunner(repository *Repository, joblogs *joblog.Broadcaster, leaseTTL time.Duration, jobs ...Job) *Runner {
	byName := make(map[string]Job, len(jobs))
	for _, job := range jobs {
		byName[job.Name()] = job
	}
	return &Runner{
		repository: repository,
		joblogs:    joblogs,
		jobs:       byName,
		instanceID: uuid.NewString(),
		leaseTTL:   leaseTTL,
		// Buffered so a burst of "run now" clicks queues up rather than
		// blocking the HTTP handler that sends them.
		trigger: make(chan string, 16),
	}
}

// Trigger queues job `name` to run as soon as the runner's loop picks it up,
// independent of its cron schedule. A full buffer drops the request rather
// than blocking the caller — the operator can just click "run now" again.
func (r *Runner) Trigger(name string) {
	select {
	case r.trigger <- name:
	default:
		config.AppLogger.WithField("job", name).Warn("scheduler trigger buffer full, dropping manual run request")
	}
}

// Start syncs scheduled_jobs with whatever was just registered, then
// launches the tick and trigger loops and returns immediately. ctx
// cancellation stops both loops.
func (r *Runner) Start(ctx context.Context) {
	defaults := make([]JobDefault, 0, len(r.jobs))
	for _, job := range r.jobs {
		defaults = append(defaults, JobDefault{Name: job.Name(), CronExpression: job.DefaultCron()})
	}
	if err := r.repository.Sync(ctx, defaults); err != nil {
		config.AppLogger.WithError(err).Error("failed to sync scheduled jobs")
	}

	go r.tickLoop(ctx)
	go r.triggerLoop(ctx)
}

// tickLoop wakes exactly on each minute boundary (recomputing the wait each
// time rather than a naive ticker) so ticks land on the minute regardless of
// when this process started.
func (r *Runner) tickLoop(ctx context.Context) {
	for {
		next := time.Now().Truncate(time.Minute).Add(time.Minute)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
			r.tick(ctx, next)
		}
	}
}

func (r *Runner) tick(ctx context.Context, minute time.Time) {
	schedules, err := r.repository.GetEnabled(ctx)
	if err != nil {
		config.AppLogger.WithError(err).Warn("failed to load enabled scheduled jobs")
		return
	}

	for _, schedule := range schedules {
		job, ok := r.jobs[schedule.Name]
		if !ok {
			// A row survives until the next Sync (next boot); a job removed
			// from code mid-process just has nothing to run its tick.
			continue
		}

		cron, err := ParseCron(schedule.CronExpression)
		if err != nil {
			config.AppLogger.WithField("job", schedule.Name).WithError(err).Warn("invalid cron expression, skipping tick")
			continue
		}
		if !cron.Matches(minute) {
			continue
		}

		claimed, err := r.repository.ClaimTick(ctx, schedule.ID, minute)
		if err != nil {
			config.AppLogger.WithField("job", schedule.Name).WithError(err).Warn("failed to claim scheduled job tick")
			continue
		}
		if !claimed {
			// Another instance's tickLoop claimed this minute first.
			continue
		}

		go r.run(ctx, schedule.ID, job, false)
	}
}

func (r *Runner) triggerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case name := <-r.trigger:
			job, ok := r.jobs[name]
			if !ok {
				continue
			}
			schedule, err := r.repository.GetByName(ctx, name)
			if err != nil {
				config.AppLogger.WithField("job", name).WithError(err).Warn("failed to load scheduled job for manual run")
				continue
			}
			go r.run(ctx, schedule.ID, job, true)
		}
	}
}

// run acquires the per-schedule lease, executes the job with a heartbeat
// keeping the lease alive for the duration, and always records the outcome
// — success, failure, or (for an automatic tick that lost the lease race)
// skipped, never silently nothing.
func (r *Runner) run(ctx context.Context, scheduleID uuid.UUID, job Job, manual bool) {
	acquired, err := r.repository.AcquireLease(ctx, scheduleID, r.instanceID, r.leaseTTL)
	if err != nil {
		config.AppLogger.WithField("job", job.Name()).WithError(err).Warn("failed to acquire scheduled job lease")
		return
	}
	if !acquired {
		if !manual {
			r.record(ctx, job.Name(), joblog.StatusSkipped, "another instance is already running this job", time.Now(), scheduleID, 0)
		}
		return
	}
	defer func() {
		if err := r.repository.ReleaseLease(context.WithoutCancel(ctx), scheduleID, r.instanceID); err != nil {
			config.AppLogger.WithField("job", job.Name()).WithError(err).Warn("failed to release scheduled job lease")
		}
	}()

	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go r.heartbeat(heartbeatCtx, scheduleID)

	started := time.Now()
	r.runOnce(ctx, job, scheduleID, started)
}

func (r *Runner) heartbeat(ctx context.Context, scheduleID uuid.UUID) {
	interval := r.leaseTTL / 3
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.repository.RenewLease(context.WithoutCancel(ctx), scheduleID, r.instanceID); err != nil {
				config.AppLogger.Warn("failed to renew scheduled job lease")
			}
		}
	}
}

// runOnce recovers a panic from job.Run so one broken job can't take down
// its own goroutine silently — without this, a panic just kills the
// goroutine and the job never runs again with nothing in go.log to explain
// why.
func (r *Runner) runOnce(ctx context.Context, job Job, scheduleID uuid.UUID, started time.Time) {
	defer func() {
		if rec := recover(); rec != nil {
			config.AppLogger.WithField("job", job.Name()).Errorf("scheduled job panicked: %v", rec)
			r.record(ctx, job.Name(), joblog.StatusFailed, fmt.Sprintf("panic: %v", rec), started, scheduleID, time.Since(started).Milliseconds())
		}
	}()

	message, err := job.Run(ctx)
	status := joblog.StatusSuccess
	if err != nil {
		status = joblog.StatusFailed
		message = err.Error()
		config.AppLogger.WithField("job", job.Name()).WithError(err).Warn("scheduled job failed")
	}

	r.record(ctx, job.Name(), status, message, started, scheduleID, time.Since(started).Milliseconds())
}

func (r *Runner) record(ctx context.Context, name string, status joblog.Status, message string, started time.Time, scheduleID uuid.UUID, durationMs int64) {
	finished := time.Now()

	// Detached from the caller's ctx with its own timeout: a canceled parent
	// (process shutting down mid-run) must not also lose the record of what
	// just happened.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	log := &joblog.JobLog{
		Type:       joblog.TypeSchedule,
		Name:       name,
		Status:     status,
		Message:    &message,
		StartedAt:  started,
		FinishedAt: &finished,
	}
	if err := r.joblogs.Record(writeCtx, log); err != nil {
		config.AppLogger.WithField("job", name).WithError(err).Warn("failed to record job log")
	}
	if err := r.repository.RecordRun(writeCtx, scheduleID, status, message, finished, durationMs); err != nil {
		config.AppLogger.WithField("job", name).WithError(err).Warn("failed to update scheduled job run state")
	}
}
