package queue

import (
	"context"
	"fmt"
	"time"
)

const (
	PruneJobsJobName    = "prune-queue-jobs"
	ReleaseStaleJobName = "release-stale-queue-jobs"
)

// PruneJobsJob is app/scheduler's job for queue_jobs — it lives here rather
// than in app/scheduler so the knowledge of what needs tidying stays with
// the thing that makes the mess.
type PruneJobsJob struct {
	repository *Repository
	retention  time.Duration
}

func NewPruneJobsJob(repository *Repository, retention time.Duration) *PruneJobsJob {
	return &PruneJobsJob{repository: repository, retention: retention}
}

func (j *PruneJobsJob) Name() string { return PruneJobsJobName }

// DefaultCron runs 30 minutes after prune-job-logs (03:00) — both are
// deletes over the two biggest tables here, so they're staggered rather
// than racing each other.
func (j *PruneJobsJob) DefaultCron() string { return "30 3 * * *" }

func (j *PruneJobsJob) Run(ctx context.Context) (string, error) {
	cutoff := time.Now().Add(-j.retention)
	n, err := j.repository.Prune(ctx, cutoff)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d queue job(s) pruned", n), nil
}

// ReleaseStaleJobsJob is a backstop for Worker.Start's own ReleaseStale
// call: if the process restarts without a clean shutdown, or a run outlives
// the process some other way, this catches the stuck job within a few
// minutes instead of leaving it in "running" until someone notices.
type ReleaseStaleJobsJob struct {
	worker *Worker
}

func NewReleaseStaleJobsJob(worker *Worker) *ReleaseStaleJobsJob {
	return &ReleaseStaleJobsJob{worker: worker}
}

func (j *ReleaseStaleJobsJob) Name() string { return ReleaseStaleJobName }

func (j *ReleaseStaleJobsJob) DefaultCron() string { return "*/5 * * * *" }

func (j *ReleaseStaleJobsJob) Run(ctx context.Context) (string, error) {
	released, failed, err := j.worker.ReleaseStale(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d job(s) released back to pending, %d failed outright", released, failed), nil
}
