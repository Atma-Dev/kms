package queue

import (
	"encoding/json"
	"api-kms/utils"
)

type Resource struct {
	ID              string          `json:"id"`
	Queue           string          `json:"queue"`
	Name            string          `json:"name"`
	Status          Status          `json:"status"`
	Attempts        int             `json:"attempts"`
	MaxAttempts     int             `json:"max_attempts"`
	Payload         json.RawMessage `json:"payload,omitempty"`
	PayloadRedacted bool            `json:"payload_redacted"`
	AvailableAt     string          `json:"available_at"`
	StartedAt       *string         `json:"started_at"`
	FinishedAt      *string         `json:"finished_at"`
	LastError       *string         `json:"last_error"`
	DurationMs      *int64          `json:"duration_ms"`
}

// NewResource redacts payload for anything Registry.IsSensitive names — a
// nil registry (some call sites don't have one handy) means "not sensitive",
// never "sensitive by default", since the safe direction for an unknown
// registry is to behave like the pre-redaction code did, not to hide every
// payload.
func NewResource(job *Job, registry *Registry) *Resource {
	resource := &Resource{
		ID:          job.ID.String(),
		Queue:       job.Queue,
		Name:        job.Name,
		Status:      job.Status,
		Attempts:    job.Attempts,
		MaxAttempts: job.MaxAttempts,
		AvailableAt: utils.FormatDateTimeVal(job.AvailableAt),
		StartedAt:   utils.FormatDateTime(job.StartedAt),
		FinishedAt:  utils.FormatDateTime(job.FinishedAt),
		LastError:   job.LastError,
	}

	if registry != nil && registry.IsSensitive(job.Name) {
		resource.PayloadRedacted = true
	} else if len(job.Payload) > 0 {
		resource.Payload = json.RawMessage(job.Payload)
	}

	if job.StartedAt != nil && job.FinishedAt != nil {
		ms := job.FinishedAt.Sub(*job.StartedAt).Milliseconds()
		resource.DurationMs = &ms
	}

	return resource
}

func NewResourceCollection(jobs []Job, registry *Registry) []*Resource {
	result := make([]*Resource, len(jobs))
	for i := range jobs {
		result[i] = NewResource(&jobs[i], registry)
	}
	return result
}
