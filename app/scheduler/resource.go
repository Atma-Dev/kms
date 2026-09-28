package scheduler

import (
	"api-kms/utils"
	"time"
)

type Resource struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	CronExpression string  `json:"cron_expression"`
	Enabled        bool    `json:"enabled"`
	NextRunAt      *string `json:"next_run_at"`
	LastRunAt      *string `json:"last_run_at"`
	LastStatus     *string `json:"last_status"`
	LastMessage    *string `json:"last_message"`
	LastDurationMs *int64  `json:"last_duration_ms"`
}

func NewResource(schedule *ScheduledJob) *Resource {
	resource := &Resource{
		ID:             schedule.ID.String(),
		Name:           schedule.Name,
		CronExpression: schedule.CronExpression,
		Enabled:        schedule.Enabled,
		LastRunAt:      utils.FormatDateTime(schedule.LastRunAt),
		LastStatus:     schedule.LastStatus,
		LastMessage:    schedule.LastMessage,
		LastDurationMs: schedule.LastDurationMs,
	}

	if schedule.Enabled {
		if cron, err := ParseCron(schedule.CronExpression); err == nil {
			if next, ok := cron.Next(time.Now()); ok {
				formatted := utils.FormatDateTimeVal(next)
				resource.NextRunAt = &formatted
			}
		}
	}

	return resource
}

func NewResourceCollection(schedules []ScheduledJob) []*Resource {
	result := make([]*Resource, len(schedules))
	for i := range schedules {
		result[i] = NewResource(&schedules[i])
	}
	return result
}
