package protection

import (
	"context"
	"fmt"
	"time"
)

const PruneViewSessionsJobName = "prune-view-sessions"

// PruneViewSessionsJob removes share links that have already expired.
//
// An expired row grants nothing — OpenSession checks the time, not the
// presence of the row — so this is housekeeping rather than a security
// control. Without it view_sessions grows by one row per link ever created
// and never shrinks, which makes the unique index longer and the per-document
// listing slower for no benefit.
//
// Only the sessions are swept. The access log rows a link produced stay: they
// are the record of what the link was used for, and that outlives the link by
// design.
type PruneViewSessionsJob struct {
	repository *Repository
}

func NewPruneViewSessionsJob(repository *Repository) *PruneViewSessionsJob {
	return &PruneViewSessionsJob{repository: repository}
}

func (j *PruneViewSessionsJob) Name() string { return PruneViewSessionsJobName }

// DefaultCron runs once a day in the quiet hours, like the other retention
// sweeps. Nothing is wrong while an expired row sits there, so there is no
// reason for this to compete with daytime traffic.
func (j *PruneViewSessionsJob) DefaultCron() string { return "50 3 * * *" }

func (j *PruneViewSessionsJob) Run(ctx context.Context) (string, error) {
	n, err := j.repository.DeleteExpiredSessions(ctx, time.Now())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d expired view link(s) pruned", n), nil
}
