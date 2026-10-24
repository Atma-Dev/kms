package scheduler

// CronExpression syntax is validated in Service.UpdateById via ParseCron,
// not a binding tag — a regex can't tell "9-17/2" from "9-99/2" the way the
// real parser already does.
type UpdateScheduleRequest struct {
	CronExpression string `json:"cron_expression" binding:"required,max=100"`
	Enabled        bool   `json:"enabled"`
}
