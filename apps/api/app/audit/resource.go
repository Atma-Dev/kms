package audit

import (
	"encoding/json"
	"api-kms/utils"
)

type TrailResource struct {
	ID          string          `json:"id"`
	ObjectTable string          `json:"object_table"`
	ObjectID    string          `json:"object_id"`
	Before      json.RawMessage `json:"before"`
	After       json.RawMessage `json:"after"`
	Tag         json.RawMessage `json:"tag"`
	Additional  json.RawMessage `json:"additional"`
}

type Resource struct {
	ID               string          `json:"id"`
	Action           string          `json:"action"`
	Message          string          `json:"message"`
	AppName          string          `json:"app_name"`
	ActorTable       string          `json:"actor_table"`
	ActorID          string          `json:"actor_id"`
	ActorName        string          `json:"actor_name"`
	ActorEmail       string          `json:"actor_email"`
	ActorPhone       string          `json:"actor_phone"`
	IPAddress        string          `json:"ip_address"`
	Method           string          `json:"method"`
	Endpoint         string          `json:"endpoint"`
	UserAgent        string          `json:"user_agent"`
	EntryObjectTable string          `json:"entry_object_table"`
	EntryObjectID    string          `json:"entry_object_id"`
	Tag              json.RawMessage `json:"tag"`
	Additional       json.RawMessage `json:"additional"`
	Trails           []TrailResource `json:"trails"`
	CreatedAt        string          `json:"created_at"`
}

// rawJSON returns valid JSON for an empty/unset column, so the client always
// gets `null` rather than an empty string that would fail to parse as JSON.
func rawJSON(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage("null")
	}
	return json.RawMessage(s)
}

func NewResource(m *Model) *Resource {
	trails := make([]TrailResource, len(m.Trails))
	for i, t := range m.Trails {
		trails[i] = TrailResource{
			ID:          t.ID.String(),
			ObjectTable: t.ObjectTable,
			ObjectID:    t.ObjectID,
			Before:      rawJSON(t.Before),
			After:       rawJSON(t.After),
			Tag:         rawJSON(t.Tag),
			Additional:  rawJSON(t.Additional),
		}
	}

	return &Resource{
		ID:               m.ID.String(),
		Action:           m.Action,
		Message:          m.Message,
		AppName:          m.AppName,
		ActorTable:       m.ActorTable,
		ActorID:          m.ActorID,
		ActorName:        m.ActorName,
		ActorEmail:       m.ActorEmail,
		ActorPhone:       m.ActorPhone,
		IPAddress:        m.IPAddress,
		Method:           m.Method,
		Endpoint:         m.Endpoint,
		UserAgent:        m.UserAgent,
		EntryObjectTable: m.EntryObjectTable,
		EntryObjectID:    m.EntryObjectID,
		Tag:              rawJSON(m.Tag),
		Additional:       rawJSON(m.Additional),
		Trails:           trails,
		CreatedAt:        utils.FormatDateTimeVal(m.CreatedAt),
	}
}

func NewResourceCollection(rows []Model) []*Resource {
	result := make([]*Resource, len(rows))
	for i := range rows {
		result[i] = NewResource(&rows[i])
	}
	return result
}
