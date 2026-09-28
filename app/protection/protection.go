// Package protection owns Modul 4: what happens to a document once it is
// readable — the watermark level it is shown at, the trail every reach for it
// leaves, and the time-limited links that let someone outside the app open it.
//
// It knows nothing about app/document, deliberately. Resolving a level takes
// the two stored values, recording an access takes a document id, and opening
// a share link returns one — none of it needs to load a document or decide who
// may read it. app/document is what calls in here, so the dependency runs one
// way and the two packages never have to import each other.
//
// What this module can and cannot do is worth stating plainly, because the
// difference decides which of these mechanisms is evidence and which is a
// deterrent:
//
//   - The PDF and the strict viewer's page images are rendered on the server
//     with the watermark drawn into the pixels. A reader cannot alter what
//     they carry, so a leaked copy names whoever exported it.
//   - The normal viewer's watermark is an overlay in the reader's own browser.
//     Anyone who opens dev tools can delete it in a few seconds. It raises the
//     cost of a careless copy; it is not proof of anything.
//   - Nothing here stops a photograph of the screen. FR-3's own wording
//     ("level pencegahan dasar") is the honest description.
package protection

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Level is FR-6: how hard a document defends itself.
//
// Stored as its string on spaces.protection_level and (nullable)
// documents.protection_level, so a row read straight from the database says
// what it is.
type Level string

const (
	// LevelNormal renders the document as itself and watermarks it in the
	// reader's browser.
	LevelNormal Level = "normal"
	// LevelStrict renders every page on the server as an image with the
	// watermark already in the pixels: there is no text in the viewer to
	// select, and the PDF is built from those same images.
	LevelStrict Level = "strict"
)

func (l Level) Valid() bool {
	return l == LevelNormal || l == LevelStrict
}

// Resolve answers which level applies to one document. A document's own level
// wins; a document that has none inherits its space, which is what the PRD's
// absent per-document row meant.
func Resolve(space Level, document *Level) Level {
	if document != nil && document.Valid() {
		return *document
	}
	if space.Valid() {
		return space
	}
	// A space seeded before this column existed, or one holding a value some
	// later migration removed. Defaulting to normal keeps documents readable;
	// defaulting to strict would silently turn every space into images on a
	// bad deploy.
	return LevelNormal
}

// Action is what the reader did with the document (FR-4).
type Action string

const (
	ActionView      Action = "view"
	ActionDownload  Action = "download"
	ActionPrint     Action = "print"
	ActionExportPDF Action = "export_pdf"
)

// ClientReportable is the one action the server cannot observe for itself.
//
// A browser print is a decision made entirely in the browser; nothing reaches
// the API. The client reports it, which means the row is a hint rather than
// evidence — a reader who suppresses the report simply does not appear. Every
// other action is recorded where it actually happens, and so cannot be
// withheld.
func (a Action) ClientReportable() bool {
	return a == ActionPrint
}

func (a Action) Valid() bool {
	switch a {
	case ActionView, ActionDownload, ActionPrint, ActionExportPDF:
		return true
	}
	return false
}

// AccessLog is one row per reach for a document's content (FR-4).
//
// It carries CreatedAt and no UpdatedAt on purpose: the table is append-only,
// and a column recording when a row last changed would be a column promising
// that rows change.
type AccessLog struct {
	ID         uuid.UUID `json:"id" gorm:"column:id;primary_key;type:uuid"`
	DocumentID uuid.UUID `json:"document_id" gorm:"column:document_id"`
	// UserID is null for a reader who arrived through a view session (FR-5) —
	// they have no account — and for one whose account was deleted after the
	// fact. The row is the evidence either way, which is why the foreign key
	// is ON DELETE SET NULL rather than the CASCADE document_views uses.
	UserID    *uuid.UUID `json:"user_id" gorm:"column:user_id"`
	Action    Action     `json:"action" gorm:"column:action"`
	IPAddress *string    `json:"ip_address" gorm:"column:ip_address"`
	UserAgent *string    `json:"user_agent" gorm:"column:user_agent"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`

	Document *documentRef `json:"document,omitempty" gorm:"foreignKey:DocumentID"`
	User     *userRef     `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (AccessLog) TableName() string {
	return "document_access_logs"
}

func (a *AccessLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// documentRef and userRef are the few columns the log listing shows beside a
// row. Preloading app/document's or app/user's own model would make this
// package import them, which is the dependency this package exists without —
// and the listing needs a title and a name, not a document and a user.
type documentRef struct {
	ID    uuid.UUID `json:"id" gorm:"column:id"`
	Title string    `json:"title" gorm:"column:title"`
	Slug  string    `json:"slug" gorm:"column:slug"`
}

func (documentRef) TableName() string { return "documents" }

type userRef struct {
	ID        uuid.UUID `json:"id" gorm:"column:id"`
	FirstName string    `json:"first_name" gorm:"column:first_name"`
	LastName  *string   `json:"last_name" gorm:"column:last_name"`
	Email     string    `json:"email" gorm:"column:email"`
}

func (userRef) TableName() string { return "users" }

// ViewSession is a time-limited link to one document (FR-5).
//
// Only the SHA-256 of the token is stored: the token itself travels in a URL,
// which ends up in browser history, a referer header, and wherever the
// recipient pastes it. A copy of this table is therefore not a set of working
// links, and the plaintext exists exactly once — in the response that created
// it.
type ViewSession struct {
	ID         uuid.UUID `json:"id" gorm:"column:id;primary_key;type:uuid"`
	DocumentID uuid.UUID `json:"document_id" gorm:"column:document_id"`
	CreatedBy  uuid.UUID `json:"created_by" gorm:"column:created_by"`
	Label      *string   `json:"label" gorm:"column:label"`
	TokenHash  string    `json:"-" gorm:"column:token_hash"`
	ExpiresAt  time.Time `json:"expires_at" gorm:"column:expires_at"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	Creator    *userRef  `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

func (ViewSession) TableName() string {
	return "view_sessions"
}

func (v *ViewSession) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

func (v *ViewSession) Expired(now time.Time) bool {
	return now.After(v.ExpiresAt)
}

// CreatorEmail names who is answerable for a share link, for the watermark on
// the copy its recipient reads. Empty when the row was loaded without its
// creator, which the watermark renders as a share with nobody's name on it
// rather than as a failure — a missing name is a worse page, not a wrong one.
func (v *ViewSession) CreatorEmail() string {
	if v.Creator == nil {
		return ""
	}
	return v.Creator.Email
}
