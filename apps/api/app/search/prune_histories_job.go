package search

import (
	"api-kms/config"
	"context"
	"fmt"
)

const PruneHistoriesJobName = "prune-search-histories"

// PruneHistoriesJob keeps search_histories from growing without end.
//
// The list only ever reads the newest ten (HistoryLimit), so everything below
// the keep mark is a row nothing will read again — one per distinct phrase
// anyone has ever typed. Nothing else removes them: the rows belong to a user
// rather than to a document, so no cascade reaches them while the account
// lives.
//
// Trimmed to a count per user rather than by age. The ceiling this exists for
// is a single account with thousands of distinct queries, and an age cutoff
// does nothing about that — while a cutoff long enough to keep a useful
// history is long enough to keep all of them.
type PruneHistoriesJob struct {
	repository *Repository
}

func NewPruneHistoriesJob(repository *Repository) *PruneHistoriesJob {
	return &PruneHistoriesJob{repository: repository}
}

func (j *PruneHistoriesJob) Name() string { return PruneHistoriesJobName }

// DefaultCron runs once a day, in the quiet hours. Nothing breaks while the
// extra rows sit there — they are unread, not wrong — so this is housekeeping
// and has no reason to compete with daytime traffic.
func (j *PruneHistoriesJob) DefaultCron() string { return "45 3 * * *" }

func (j *PruneHistoriesJob) Run(ctx context.Context) (string, error) {
	n, err := j.repository.PruneHistories(ctx, config.AppConfig.SearchHistoryKeepPerUser)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d search history row(s) pruned", n), nil
}
