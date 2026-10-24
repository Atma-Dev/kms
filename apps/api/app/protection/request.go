package protection

import (
	"api-kms/errors"
	"time"

	"github.com/gin-gonic/gin"
)

// SetProtectionLevelRequest sets a space's default or a document's override
// (FR-6).
//
// The pointer is what carries "inherit the space" on a document: an absent
// protection_level clears the override, which the level itself cannot express
// — "normal" is a decision, and inheriting is the absence of one. A space has
// nothing to inherit from, so its handler refuses a null.
type SetProtectionLevelRequest struct {
	ProtectionLevel *Level `form:"protection_level" json:"protection_level" binding:"omitempty,oneof=normal strict"`
}

// CreateViewSessionRequest is the share link form (FR-5).
//
// Hours rather than a date: the question anyone shares a document with is "for
// how long", and an expiry date sent by a client in an unknown timezone is a
// link that dies a day early for half the people who make one.
type CreateViewSessionRequest struct {
	Label          *string `form:"label" json:"label" binding:"omitempty,max=255"`
	ExpiresInHours int     `form:"expires_in_hours" json:"expires_in_hours" binding:"required,min=1"`
}

func (r CreateViewSessionRequest) TTL() time.Duration {
	return time.Duration(r.ExpiresInHours) * time.Hour
}

// ReportAccessRequest is the one access the client tells us about (FR-4).
//
// Only print is accepted. Every other action is observed by the server where
// it happens, and an endpoint that took any of them would let a client write
// rows describing things that never occurred — a trail anyone can forge into
// is worse than one with a known gap in it.
type ReportAccessRequest struct {
	Action Action `form:"action" json:"action" binding:"required,oneof=print"`
}

// ParseAccessLogFilter reads the trail's query string.
//
// The date parsing mirrors app/search's: the same format, and `to` covering
// the whole of its day. A range ending "today" that excluded everything that
// happened today is the reliably wrong answer, and an investigation is exactly
// where that would be noticed last.
func ParseAccessLogFilter(c *gin.Context) (AccessLogFilter, error) {
	filter := AccessLogFilter{}

	if action := Action(c.Query("action")); action != "" {
		if !action.Valid() {
			return filter, errors.InvalidStatusException("unknown action").
				WithErrors(map[string][]string{"action": {"must be view, download, print or export_pdf"}})
		}
		filter.Action = action
	}

	from, err := parseDate(c.Query("from"), "from")
	if err != nil {
		return filter, err
	}
	to, err := parseDate(c.Query("to"), "to")
	if err != nil {
		return filter, err
	}
	if to != nil {
		end := to.AddDate(0, 0, 1)
		to = &end
	}

	filter.From = from
	filter.To = to
	return filter, nil
}

// parseDate is app/search's, copied rather than shared. The obvious home is
// utils, which cannot import api-kms/errors (that package imports utils), so
// a shared version could not return the field-level validation error either
// caller wants. Ten lines twice beats a helper that has to be worse at its job.
func parseDate(value, field string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, errors.InvalidStatusException("date filters must look like 2006-01-02").
			WithErrors(map[string][]string{field: {"must look like 2006-01-02"}}).
			WithCause(err)
	}
	return &parsed, nil
}
