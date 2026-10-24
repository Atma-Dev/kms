package notification

import (
	"encoding/json"
	"api-kms/app"
	"time"
)

// Notification types this app currently emits. Kept as plain strings (not an
// enum table) since the frontend only switches on them for icon/label choice.
const (
	TypeInvoiceStatusChanged = "invoice.status_changed"
)

// Notification is one event delivered to one user (§ in-app notification
// feed). data carries whatever the frontend needs to deep-link into the
// event (invoice_number, old/new status, ...) without a column per type.
type Notification struct {
	app.BaseModel `gorm:"embedded"`
	UserID        string          `json:"user_id" gorm:"column:user_id"`
	Type          string          `json:"type" gorm:"column:type"`
	Title         string          `json:"title" gorm:"column:title"`
	Body          string          `json:"body" gorm:"column:body"`
	Data          json.RawMessage `json:"data" gorm:"column:data;type:json"`
	ReadAt        *time.Time      `json:"read_at" gorm:"column:read_at"`
}

func (Notification) TableName() string {
	return "notifications"
}
