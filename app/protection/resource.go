package protection

import (
	"api-kms/utils"
	"strings"
)

// AccessLogResource is one row of the trail as an investigator reads it: what
// happened, to which document, by whom, from where (FR-4).
//
// The document title and the reader's name are flattened in rather than nested
// as objects. This is a table, one row per line, and a client that has to
// reach into three levels of JSON to print a line ends up doing it differently
// on every screen that shows one.
type AccessLogResource struct {
	ID         string  `json:"id"`
	DocumentID string  `json:"document_id"`
	Document   string  `json:"document"`
	UserID     *string `json:"user_id"`
	// User is "—" for an access through a share link (nobody was logged in)
	// and for one whose account has since been deleted. The row survives its
	// reader on purpose; see the migration.
	User      string  `json:"user"`
	UserEmail string  `json:"user_email"`
	Action    Action  `json:"action"`
	IPAddress *string `json:"ip_address"`
	UserAgent *string `json:"user_agent"`
	CreatedAt string  `json:"created_at"`
}

func NewAccessLogResource(l *AccessLog) AccessLogResource {
	resource := AccessLogResource{
		ID:         l.ID.String(),
		DocumentID: l.DocumentID.String(),
		Action:     l.Action,
		IPAddress:  l.IPAddress,
		UserAgent:  l.UserAgent,
		CreatedAt:  utils.FormatDateTimeVal(l.CreatedAt),
	}
	if l.Document != nil {
		resource.Document = l.Document.Title
	}
	if l.UserID != nil {
		id := l.UserID.String()
		resource.UserID = &id
	}
	if l.User != nil {
		resource.User = l.User.fullName()
		resource.UserEmail = l.User.Email
	}
	return resource
}

func NewAccessLogResourceCollection(logs []AccessLog) []AccessLogResource {
	result := make([]AccessLogResource, len(logs))
	for i := range logs {
		result[i] = NewAccessLogResource(&logs[i])
	}
	return result
}

func (u *userRef) fullName() string {
	return strings.TrimSpace(u.FirstName + " " + utils.Deref(u.LastName))
}

// ViewSessionResource is a live share link as its creator sees it in the list
// (FR-5). No token: the plaintext was shown once, at creation, and this row
// only holds its hash.
type ViewSessionResource struct {
	ID         string  `json:"id"`
	DocumentID string  `json:"document_id"`
	Label      *string `json:"label"`
	CreatedBy  string  `json:"created_by"`
	Creator    string  `json:"creator"`
	ExpiresAt  string  `json:"expires_at"`
	CreatedAt  string  `json:"created_at"`
}

func NewViewSessionResource(v *ViewSession) ViewSessionResource {
	resource := ViewSessionResource{
		ID:         v.ID.String(),
		DocumentID: v.DocumentID.String(),
		Label:      v.Label,
		CreatedBy:  v.CreatedBy.String(),
		ExpiresAt:  utils.FormatDateTimeVal(v.ExpiresAt),
		CreatedAt:  utils.FormatDateTimeVal(v.CreatedAt),
	}
	if v.Creator != nil {
		resource.Creator = v.Creator.fullName()
	}
	return resource
}

func NewViewSessionResourceCollection(sessions []ViewSession) []ViewSessionResource {
	result := make([]ViewSessionResource, len(sessions))
	for i := range sessions {
		result[i] = NewViewSessionResource(&sessions[i])
	}
	return result
}

// CreatedViewSessionResource is the one response that carries the token, and
// the only chance anyone has to copy it.
type CreatedViewSessionResource struct {
	ViewSessionResource
	// Token is the plaintext, shown once. The client turns it into a URL —
	// building the link here would mean this package knowing the frontend's
	// route, which is the frontend's to change.
	Token string `json:"token"`
}

func NewCreatedViewSessionResource(v *ViewSession, token string) CreatedViewSessionResource {
	return CreatedViewSessionResource{
		ViewSessionResource: NewViewSessionResource(v),
		Token:               token,
	}
}
