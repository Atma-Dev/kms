package activitylog

import (
	"api-kms/app"
	"api-kms/app/user"
)

// ActivityLog is one HTTP request made against the API, written by
// middleware.LoggerMiddleware after every request completes. It's a
// low-effort audit trail (who hit which endpoint, when, with what result)
// rather than a per-action description — see repository-level logging if a
// specific mutation needs a richer "what changed" record.
type ActivityLog struct {
	app.BaseModel `gorm:"embedded"`
	UserID        *string    `json:"user_id" gorm:"column:user_id"`
	User          *user.User `json:"user" gorm:"foreignKey:UserID;references:ID"`
	Method        string     `json:"method" gorm:"column:method"`
	Path          string     `json:"path" gorm:"column:path"`
	StatusCode    int        `json:"status_code" gorm:"column:status_code"`
	IP            string     `json:"ip" gorm:"column:ip"`
	// Redacted and truncated by SanitizeBody before this struct is ever
	// filled in — never write a raw body into these fields directly.
	RequestBody  *string `json:"request_body" gorm:"column:request_body;type:text"`
	ResponseBody *string `json:"response_body" gorm:"column:response_body;type:text"`
}

func (ActivityLog) TableName() string {
	return "activity_logs"
}
