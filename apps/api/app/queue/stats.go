package queue

import (
	"context"
	"api-kms/errors"
	"time"
)

// Stats is "how backed up is the queue right now" — a snapshot of current
// state, independent of the paginated job list. RegisteredJobs/Queues are
// filled in by Service.GetStats, which is the layer that knows about the
// Registry.
type Stats struct {
	ByStatus                map[Status]int64 `json:"by_status"`
	DelayedCount            int64            `json:"delayed_count"`
	OldestPendingAgeSeconds *int64           `json:"oldest_pending_age_seconds"`
	ByQueue                 []QueueCount     `json:"by_queue"`
	ByName                  []NameCount      `json:"by_name"`
	RegisteredJobs          []string         `json:"registered_jobs"`
	Queues                  []string         `json:"queues"`
}

type QueueCount struct {
	Queue string `json:"queue"`
	Count int64  `json:"count"`
}

type NameCount struct {
	Name    string `json:"name"`
	Count   int64  `json:"count"`
	Pending int64  `json:"pending"`
	Running int64  `json:"running"`
	Success int64  `json:"success"`
	Failed  int64  `json:"failed"`
}

func (r *Repository) GetStats(ctx context.Context) (*Stats, error) {
	var statusRows []struct {
		Status Status
		Count  int64
	}
	if err := r.db.WithContext(ctx).Model(&Job{}).Select("status, COUNT(*) as count").Group("status").Scan(&statusRows).Error; err != nil {
		return nil, errors.InternalServerError("failed to summarize queue jobs").WithCause(err)
	}
	byStatus := make(map[Status]int64, len(statusRows))
	for _, row := range statusRows {
		byStatus[row.Status] = row.Count
	}

	var delayedCount int64
	if err := r.db.WithContext(ctx).Model(&Job{}).
		Where("status = ? AND available_at > ?", StatusPending, time.Now()).
		Count(&delayedCount).Error; err != nil {
		return nil, errors.InternalServerError("failed to count delayed queue jobs").WithCause(err)
	}

	var oldestRow struct{ AvailableAt *time.Time }
	if err := r.db.WithContext(ctx).Model(&Job{}).
		Where("status = ?", StatusPending).
		Select("MIN(available_at) as available_at").
		Scan(&oldestRow).Error; err != nil {
		return nil, errors.InternalServerError("failed to find oldest pending queue job").WithCause(err)
	}

	var byQueue []QueueCount
	if err := r.db.WithContext(ctx).Model(&Job{}).
		Select("queue, COUNT(*) as count").Group("queue").Scan(&byQueue).Error; err != nil {
		return nil, errors.InternalServerError("failed to aggregate queue jobs by queue").WithCause(err)
	}

	var byName []NameCount
	if err := r.db.WithContext(ctx).Model(&Job{}).
		Select(`name, COUNT(*) as count,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'running') as running,
			COUNT(*) FILTER (WHERE status = 'success') as success,
			COUNT(*) FILTER (WHERE status = 'failed') as failed`).
		Group("name").Scan(&byName).Error; err != nil {
		return nil, errors.InternalServerError("failed to aggregate queue jobs by name").WithCause(err)
	}

	stats := &Stats{ByStatus: byStatus, DelayedCount: delayedCount, ByQueue: byQueue, ByName: byName}
	if oldestRow.AvailableAt != nil {
		age := int64(time.Since(*oldestRow.AvailableAt).Seconds())
		stats.OldestPendingAgeSeconds = &age
	}
	return stats, nil
}
