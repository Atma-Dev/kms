package joblog

import (
	"context"
	"fmt"
	"api-kms/errors"
	"api-kms/utils"
	"time"

	"gorm.io/gorm"
)

// Stats answers "how has background work been doing lately" — the dashboard
// half of monitoring, independent of the paginated run-history list.
type Stats struct {
	Summary Summary      `json:"summary"`
	Trend   []TrendPoint `json:"trend"`
	ByJob   []JobStat    `json:"by_job"`
}

type Summary struct {
	Total         int64   `json:"total"`
	Success       int64   `json:"success"`
	Failed        int64   `json:"failed"`
	Skipped       int64   `json:"skipped"`
	SuccessRate   float64 `json:"success_rate"`
	AvgDurationMs int64   `json:"avg_duration_ms"`
	MaxDurationMs int64   `json:"max_duration_ms"`
}

// TrendPoint is one bucket of a time series. Bucket is a pre-formatted label
// (hour or day) rather than a raw timestamp so the frontend never has to
// re-derive the bucketing rule the backend used to build it.
type TrendPoint struct {
	Bucket  string `json:"bucket"`
	Success int64  `json:"success"`
	Failed  int64  `json:"failed"`
	Skipped int64  `json:"skipped"`
}

type JobStat struct {
	Type          Type    `json:"type"`
	Name          string  `json:"name"`
	Total         int64   `json:"total"`
	Success       int64   `json:"success"`
	Failed        int64   `json:"failed"`
	Skipped       int64   `json:"skipped"`
	AvgDurationMs int64   `json:"avg_duration_ms"`
	MaxDurationMs int64   `json:"max_duration_ms"`
	LastStatus    *Status `json:"last_status"`
	LastRunAt     *string `json:"last_run_at"`
	FLastRunAt    *string `json:"f_last_run_at"`
}

// window bundles the range query param's derived shape: how far back to
// look, how to format a bucket label, and how big a step advances one bucket
// — kept together so trend()'s gap-fill loop and the SQL DATE_FORMAT pattern
// can never drift apart from each other.
type window struct {
	since      time.Time
	dateFormat string // MySQL DATE_FORMAT pattern
	goLayout   string // equivalent Go time layout, for the gap-fill loop
	step       time.Duration
}

func newWindow(rangeParam string) window {
	now := time.Now()
	switch rangeParam {
	case "7d":
		return window{since: now.Add(-7 * 24 * time.Hour), dateFormat: "%Y-%m-%d", goLayout: "2006-01-02", step: 24 * time.Hour}
	case "30d":
		return window{since: now.Add(-30 * 24 * time.Hour), dateFormat: "%Y-%m-%d", goLayout: "2006-01-02", step: 24 * time.Hour}
	default: // "24h" and anything unrecognized
		return window{since: now.Add(-24 * time.Hour), dateFormat: "%Y-%m-%d %H:00", goLayout: "2006-01-02 15:00", step: time.Hour}
	}
}

// scope applies the range/type filter every aggregation below shares.
func (r *Repository) scope(ctx context.Context, since time.Time, jobType string) *gorm.DB {
	db := r.db.WithContext(ctx).Model(&JobLog{}).Where("started_at >= ?", since)
	if jobType != "" {
		db = db.Where("type = ?", jobType)
	}
	return db
}

// GetStats aggregates job_logs for the dashboard. rangeParam is one of
// 24h|7d|30d (default 24h); jobType optionally narrows to "schedule" or
// "queue".
func (r *Repository) GetStats(ctx context.Context, rangeParam, jobType string) (*Stats, error) {
	w := newWindow(rangeParam)

	summary, err := r.summary(ctx, w.since, jobType)
	if err != nil {
		return nil, err
	}
	trend, err := r.trend(ctx, w, jobType)
	if err != nil {
		return nil, err
	}
	byJob, err := r.byJob(ctx, w.since, jobType)
	if err != nil {
		return nil, err
	}

	return &Stats{Summary: *summary, Trend: trend, ByJob: byJob}, nil
}

func (r *Repository) summary(ctx context.Context, since time.Time, jobType string) (*Summary, error) {
	var counts []struct {
		Status Status
		Count  int64
	}
	if err := r.scope(ctx, since, jobType).Select("status, COUNT(*) as count").Group("status").Scan(&counts).Error; err != nil {
		return nil, errors.InternalServerError("failed to summarize job logs").WithCause(err)
	}

	summary := &Summary{}
	for _, row := range counts {
		summary.Total += row.Count
		switch row.Status {
		case StatusSuccess:
			summary.Success = row.Count
		case StatusFailed:
			summary.Failed = row.Count
		case StatusSkipped:
			summary.Skipped = row.Count
		}
	}
	if finished := summary.Success + summary.Failed; finished > 0 {
		summary.SuccessRate = float64(summary.Success) / float64(finished) * 100
	}

	var duration struct {
		AvgMs *float64
		MaxMs *float64
	}
	err := r.scope(ctx, since, jobType).
		Where("finished_at IS NOT NULL").
		Select("AVG(TIMESTAMPDIFF(MICROSECOND, started_at, finished_at)) / 1000 as avg_ms, MAX(TIMESTAMPDIFF(MICROSECOND, started_at, finished_at)) / 1000 as max_ms").
		Scan(&duration).Error
	if err != nil {
		return nil, errors.InternalServerError("failed to summarize job log durations").WithCause(err)
	}
	if duration.AvgMs != nil {
		summary.AvgDurationMs = int64(*duration.AvgMs)
	}
	if duration.MaxMs != nil {
		summary.MaxDurationMs = int64(*duration.MaxMs)
	}

	return summary, nil
}

func (r *Repository) trend(ctx context.Context, w window, jobType string) ([]TrendPoint, error) {
	var rows []struct {
		Bucket string
		Status Status
		Count  int64
	}
	err := r.scope(ctx, w.since, jobType).
		Select(fmt.Sprintf("DATE_FORMAT(started_at, '%s') as bucket, status, COUNT(*) as count", w.dateFormat)).
		Group("bucket, status").
		Scan(&rows).Error
	if err != nil {
		return nil, errors.InternalServerError("failed to build job log trend").WithCause(err)
	}

	byBucket := make(map[string]*TrendPoint, len(rows))
	for _, row := range rows {
		point, ok := byBucket[row.Bucket]
		if !ok {
			point = &TrendPoint{Bucket: row.Bucket}
			byBucket[row.Bucket] = point
		}
		switch row.Status {
		case StatusSuccess:
			point.Success = row.Count
		case StatusFailed:
			point.Failed = row.Count
		case StatusSkipped:
			point.Skipped = row.Count
		}
	}

	// Gap-fill: a bucket with zero runs must still appear as zero, or an
	// outage (nothing ran) renders identically to "no data for this chart".
	trend := make([]TrendPoint, 0)
	now := time.Now()
	for t := w.since.Truncate(w.step); !t.After(now); t = t.Add(w.step) {
		label := t.Format(w.goLayout)
		if point, ok := byBucket[label]; ok {
			trend = append(trend, *point)
		} else {
			trend = append(trend, TrendPoint{Bucket: label})
		}
	}
	return trend, nil
}

func (r *Repository) byJob(ctx context.Context, since time.Time, jobType string) ([]JobStat, error) {
	whereSQL := "started_at >= ?"
	args := []any{since}
	if jobType != "" {
		whereSQL += " AND type = ?"
		args = append(args, jobType)
	}

	var totals []struct {
		Type    Type
		Name    string
		Total   int64
		Success int64
		Failed  int64
		Skipped int64
		AvgMs   *float64
		MaxMs   *float64
	}
	err := r.db.WithContext(ctx).Model(&JobLog{}).
		Where(whereSQL, args...).
		Select(`type, name, COUNT(*) as total,
			SUM(status = 'success') as success,
			SUM(status = 'failed') as failed,
			SUM(status = 'skipped') as skipped,
			AVG(CASE WHEN finished_at IS NOT NULL THEN TIMESTAMPDIFF(MICROSECOND, started_at, finished_at) / 1000 END) as avg_ms,
			MAX(CASE WHEN finished_at IS NOT NULL THEN TIMESTAMPDIFF(MICROSECOND, started_at, finished_at) / 1000 END) as max_ms`).
		Group("type, name").
		Scan(&totals).Error
	if err != nil {
		return nil, errors.InternalServerError("failed to aggregate job logs by job").WithCause(err)
	}

	// Last-run status per job, scoped by the same window/type filter — a
	// window-function rank-and-filter rather than one query per job name.
	var latest []struct {
		Type      Type
		Name      string
		Status    Status
		StartedAt time.Time
	}
	rawSQL := fmt.Sprintf(`
		SELECT type, name, status, started_at FROM (
			SELECT type, name, status, started_at,
			       ROW_NUMBER() OVER (PARTITION BY type, name ORDER BY started_at DESC) AS rn
			FROM job_logs WHERE %s
		) ranked WHERE rn = 1
	`, whereSQL)
	if err := r.db.WithContext(ctx).Raw(rawSQL, args...).Scan(&latest).Error; err != nil {
		return nil, errors.InternalServerError("failed to find last job log status").WithCause(err)
	}

	last := make(map[string]struct {
		Status    Status
		StartedAt time.Time
	}, len(latest))
	for _, row := range latest {
		key := string(row.Type) + "|" + row.Name
		last[key] = struct {
			Status    Status
			StartedAt time.Time
		}{row.Status, row.StartedAt}
	}

	result := make([]JobStat, len(totals))
	for i, row := range totals {
		result[i] = JobStat{
			Type: row.Type, Name: row.Name, Total: row.Total,
			Success: row.Success, Failed: row.Failed, Skipped: row.Skipped,
		}
		if row.AvgMs != nil {
			result[i].AvgDurationMs = int64(*row.AvgMs)
		}
		if row.MaxMs != nil {
			result[i].MaxDurationMs = int64(*row.MaxMs)
		}
		if l, ok := last[string(row.Type)+"|"+row.Name]; ok {
			status := l.Status
			startedAt := l.StartedAt
			startedAtStr := startedAt.Format(time.RFC3339)
			result[i].LastStatus = &status
			result[i].LastRunAt = &startedAtStr
			result[i].FLastRunAt = utils.FormatDateTime(&startedAt)
		}
	}
	return result, nil
}
