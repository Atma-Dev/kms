package scheduler

import (
	"context"
	"fmt"
	"api-kms/app/joblog"
	"api-kms/config"
	"time"
)

const PruneJobLogsJobName = "prune-job-logs"

// PruneJobLogsJob keeps job_logs from growing forever — the two schedule
// jobs alone write ~2,880 rows/day. Runs once a day rather than every
// minute: retention is measured in days, so minute-level granularity buys
// nothing and would only add this job's own bookkeeping rows faster.
type PruneJobLogsJob struct {
	repository *joblog.Repository
}

func NewPruneJobLogsJob(repository *joblog.Repository) *PruneJobLogsJob {
	return &PruneJobLogsJob{repository: repository}
}

func (j *PruneJobLogsJob) Name() string { return PruneJobLogsJobName }

// DefaultCron runs once a day at 03:00 rather than every minute: retention
// is measured in days, so minute-level granularity buys nothing and would
// only add this job's own bookkeeping rows faster.
func (j *PruneJobLogsJob) DefaultCron() string { return "0 3 * * *" }

func (j *PruneJobLogsJob) Run(ctx context.Context) (string, error) {
	retention := time.Duration(config.AppConfig.ScheduleJobLogRetentionDays) * 24 * time.Hour
	cutoff := time.Now().Add(-retention)

	n, err := j.repository.Prune(ctx, cutoff)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d job log(s) pruned", n), nil
}
