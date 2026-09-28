package notification

import (
	"encoding/json"
	"api-kms/utils"
	"time"
)

// Raw values plus f_ formatted twins, same convention as everywhere else
// (e.g. invoice.Resource): the client sorts/groups on the raw field and
// renders the f_ one.
type Resource struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	Data       json.RawMessage `json:"data"`
	IsRead     bool            `json:"is_read"`
	CreatedAt  time.Time       `json:"created_at"`
	FCreatedAt string          `json:"f_created_at"`
}

func NewResource(n *Notification) Resource {
	return Resource{
		ID:         n.ID.String(),
		Type:       n.Type,
		Title:      n.Title,
		Body:       n.Body,
		Data:       n.Data,
		IsRead:     n.ReadAt != nil,
		CreatedAt:  n.CreatedAt,
		FCreatedAt: utils.FormatDateTimeVal(n.CreatedAt),
	}
}

func NewResourceCollection(notifications []Notification) []Resource {
	result := make([]Resource, len(notifications))
	for i := range notifications {
		result[i] = NewResource(&notifications[i])
	}
	return result
}
