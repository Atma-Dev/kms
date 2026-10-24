package joblog

import "api-kms/utils"

type Resource struct {
	ID         string  `json:"id"`
	Type       Type    `json:"type"`
	Name       string  `json:"name"`
	Status     Status  `json:"status"`
	Message    *string `json:"message"`
	StartedAt  string  `json:"started_at"`
	FinishedAt *string `json:"finished_at"`
}

func NewResource(log *JobLog) *Resource {
	return &Resource{
		ID:         log.ID.String(),
		Type:       log.Type,
		Name:       log.Name,
		Status:     log.Status,
		Message:    log.Message,
		StartedAt:  utils.FormatDateTimeVal(log.StartedAt),
		FinishedAt: utils.FormatDateTime(log.FinishedAt),
	}
}

func NewResourceCollection(logs []JobLog) []*Resource {
	result := make([]*Resource, len(logs))
	for i := range logs {
		result[i] = NewResource(&logs[i])
	}
	return result
}
