package queue

import (
	"context"
	"fmt"
	"api-kms/app/joblog"
	"api-kms/config"
	"time"
)

type WorkerConfig struct {
	Concurrency  int
	Queues       []string
	PollInterval time.Duration
	MaxRunTime   time.Duration
	BackoffBase  time.Duration
	BackoffMax   time.Duration
	StaleAfter   time.Duration
}

func (c WorkerConfig) withDefaults() WorkerConfig {
	if c.Concurrency <= 0 {
		c.Concurrency = 5
	}
	if len(c.Queues) == 0 {
		c.Queues = []string{DefaultQueue}
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 3 * time.Second
	}
	if c.MaxRunTime <= 0 {
		c.MaxRunTime = 5 * time.Minute
	}
	if c.BackoffBase <= 0 {
		c.BackoffBase = 10 * time.Second
	}
	if c.BackoffMax <= 0 {
		c.BackoffMax = 10 * time.Minute
	}
	if c.StaleAfter <= 0 {
		c.StaleAfter = c.MaxRunTime * 2
	}
	return c
}

// Worker runs Concurrency independent poll loops, each claiming and
// processing jobs until its queues run dry, then waiting for either the
// next poll tick or a Redis doorbell.
type Worker struct {
	repository *Repository
	registry   *Registry
	joblogs    *joblog.Broadcaster
	hub        *Hub
	notifier   *Notifier
	config     WorkerConfig
}

func NewWorker(repository *Repository, registry *Registry, joblogs *joblog.Broadcaster, hub *Hub, notifier *Notifier, cfg WorkerConfig) *Worker {
	return &Worker{repository: repository, registry: registry, joblogs: joblogs, hub: hub, notifier: notifier, config: cfg.withDefaults()}
}

// Start recovers anything a previous process left running, then launches
// the poll loops and returns immediately.
func (w *Worker) Start(ctx context.Context) {
	if _, _, err := w.ReleaseStale(ctx); err != nil {
		config.AppLogger.WithError(err).Warn("failed to release stale queue jobs at startup")
	}

	wake := w.notifier.Listen(ctx)
	for i := 0; i < w.config.Concurrency; i++ {
		go w.loop(ctx, wake)
	}
}

func (w *Worker) loop(ctx context.Context, wake <-chan string) {
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		w.drain(ctx)
		select {
		case <-ctx.Done():
			return
		case <-wake:
		case <-ticker.C:
		}
	}
}

// drain claims and processes jobs until the queue is empty. Several
// goroutines drain concurrently (one per Concurrency), which is exactly
// what Repository.Claim's SKIP LOCKED protects against double-claiming.
func (w *Worker) drain(ctx context.Context) {
	for {
		job, err := w.repository.Claim(ctx, w.config.Queues)
		if err != nil {
			config.AppLogger.WithError(err).Warn("failed to claim queue job")
			return
		}
		if job == nil {
			return
		}
		w.hub.Broadcast(NewResource(job, w.registry))
		w.process(ctx, job)
	}
}

func (w *Worker) process(ctx context.Context, job *Job) {
	started := time.Now()
	var message string
	var runErr error

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				runErr = fmt.Errorf("panic: %v", rec)
			}
		}()

		handler, ok := w.registry.Get(job.Name)
		if !ok {
			runErr = Permanent(fmt.Errorf("no queue handler is registered for %s", job.Name))
			return
		}

		runCtx, cancel := context.WithTimeout(ctx, w.config.MaxRunTime)
		defer cancel()
		message, runErr = handler.Handle(runCtx, job.Payload)
	}()

	w.finish(ctx, job, started, message, runErr)
}

func (w *Worker) finish(ctx context.Context, job *Job, started time.Time, message string, runErr error) {
	// Detached with its own timeout: a canceled parent (process shutting
	// down mid-run) must not also lose the record of what just happened.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	finished := time.Now()
	logStatus := joblog.StatusSuccess

	if runErr != nil {
		logStatus = joblog.StatusFailed
		message = runErr.Error()
		config.AppLogger.WithField("job", job.Name).WithError(runErr).Warn("queue job failed")

		exhausted := job.Attempts >= job.MaxAttempts || IsPermanent(runErr)
		var retryAt *time.Time
		if !exhausted {
			at := finished.Add(Backoff(job.Attempts, w.config.BackoffBase, w.config.BackoffMax))
			retryAt = &at
			job.Status = StatusPending
		} else {
			job.Status = StatusFailed
		}
		if err := w.repository.MarkFailed(writeCtx, job, message, finished, retryAt); err != nil {
			config.AppLogger.WithField("job", job.Name).WithError(err).Warn("failed to record queue job failure")
		}
	} else {
		job.Status = StatusSuccess
		if err := w.repository.MarkSucceeded(writeCtx, job, finished); err != nil {
			config.AppLogger.WithField("job", job.Name).WithError(err).Warn("failed to record queue job success")
		}
	}
	job.FinishedAt = &finished

	if job.Status.IsFinished() && w.registry.IsSensitive(job.Name) {
		if err := w.repository.ClearPayload(writeCtx, job.ID); err != nil {
			config.AppLogger.WithField("job", job.Name).WithError(err).Warn("failed to clear sensitive queue job payload")
		}
		job.Payload = nil
	}

	logMessage := message
	log := &joblog.JobLog{
		Type:       joblog.TypeQueue,
		Name:       job.Name,
		Status:     logStatus,
		Message:    &logMessage,
		StartedAt:  started,
		FinishedAt: &finished,
	}
	if err := w.joblogs.Record(writeCtx, log); err != nil {
		config.AppLogger.WithField("job", job.Name).WithError(err).Warn("failed to record job log")
	}

	w.hub.Broadcast(NewResource(job, w.registry))
}

// ReleaseStale recovers jobs a worker died mid-run holding. Runs at
// Worker.Start and again every few minutes as its own scheduled job (see
// ReleaseStaleJobsJob) — a crash between the two shouldn't strand a job in
// "running" indefinitely.
func (w *Worker) ReleaseStale(ctx context.Context) (released, failed int64, err error) {
	return w.repository.ReleaseStale(ctx, w.config.StaleAfter)
}
