package activitylog

import (
	"api-kms/app/user"
	"api-kms/utils"
)

type Resource struct {
	ID           string         `json:"id"`
	User         *user.Resource `json:"user"`
	Method       string         `json:"method"`
	Path         string         `json:"path"`
	StatusCode   int            `json:"status_code"`
	IP           string         `json:"ip"`
	RequestBody  *string        `json:"request_body"`
	ResponseBody *string        `json:"response_body"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
}

func NewResource(log *ActivityLog) *Resource {
	return &Resource{
		ID:           log.ID.String(),
		User:         user.NewResource(log.User),
		Method:       log.Method,
		Path:         log.Path,
		StatusCode:   log.StatusCode,
		IP:           log.IP,
		RequestBody:  log.RequestBody,
		ResponseBody: log.ResponseBody,
		CreatedAt:    utils.FormatDateTimeVal(log.CreatedAt),
		UpdatedAt:    utils.FormatDateTimeVal(log.UpdatedAt),
	}
}

func NewResourceCollection(logs []ActivityLog) []*Resource {
	result := make([]*Resource, len(logs))
	for i := range logs {
		result[i] = NewResource(&logs[i])
	}
	return result
}
