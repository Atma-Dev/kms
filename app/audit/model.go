package audit

import "api-kms/app"

// Model is the GORM row for one audited action (who, from where, on what).
type Model struct {
	app.BaseModel    `gorm:"embedded"`
	Action           string       `json:"action" gorm:"column:action"`
	Message          string       `json:"message" gorm:"column:message"`
	AppName          string       `json:"app_name" gorm:"column:app_name"`
	ActorTable       string       `json:"actor_table" gorm:"column:actor_table"`
	ActorID          string       `json:"actor_id" gorm:"column:actor_id"`
	ActorName        string       `json:"actor_name" gorm:"column:actor_name"`
	ActorEmail       string       `json:"actor_email" gorm:"column:actor_email"`
	ActorPhone       string       `json:"actor_phone" gorm:"column:actor_phone"`
	IPAddress        string       `json:"ip_address" gorm:"column:ip_address"`
	Method           string       `json:"method" gorm:"column:method"`
	Endpoint         string       `json:"endpoint" gorm:"column:endpoint"`
	UserAgent        string       `json:"user_agent" gorm:"column:user_agent"`
	EntryObjectTable string       `json:"entry_object_table" gorm:"column:entry_object_table"`
	EntryObjectID    string       `json:"entry_object_id" gorm:"column:entry_object_id"`
	Tag              string       `json:"tag" gorm:"column:tag"`
	Additional       string       `json:"additional" gorm:"column:additional"`
	Trails           []TrailModel `json:"trails" gorm:"foreignKey:AuditID"`
}

func (Model) TableName() string { return "audits" }

// TrailModel is one changed record captured within an audit.
type TrailModel struct {
	app.BaseModel `gorm:"embedded"`
	AuditID       string `json:"audit_id" gorm:"column:audit_id"`
	ObjectTable   string `json:"object_table" gorm:"column:object_table"`
	ObjectID      string `json:"object_id" gorm:"column:object_id"`
	Before        string `json:"before" gorm:"column:before"`
	After         string `json:"after" gorm:"column:after"`
	Tag           string `json:"tag" gorm:"column:tag"`
	Additional    string `json:"additional" gorm:"column:additional"`
}

func (TrailModel) TableName() string { return "audit_trails" }
