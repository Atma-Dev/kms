package queue

// RetryFailedRequest optionally narrows a bulk retry to one job name; empty
// means every failed job.
type RetryFailedRequest struct {
	Name string `json:"name" binding:"omitempty,max=100"`
}

// PurgeRequest deliberately excludes pending/running from the allowed
// values — purging only ever means "clear finished rows I don't need
// anymore".
type PurgeRequest struct {
	Statuses []Status `json:"statuses" binding:"required,min=1,dive,oneof=success failed canceled"`
}
