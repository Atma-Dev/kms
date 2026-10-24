// Package document owns the knowledge itself: the document, its lifecycle, and
// the version snapshots taken as it changes.
//
// Versions live here rather than in a package of their own for the same reason
// sections live inside a course: a version is only ever reachable through the
// document that owns it, and every write to one is part of a write to the
// other — splitting them would put the two halves of a single transaction in
// two packages.
package document

import (
	"api-kms/app"
	"api-kms/app/protection"
	"database/sql/driver"
	"encoding/json"
	stderrors "errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Status is the lifecycle in FR-4. Stored as its string, not an int: a row
// read straight from the database says what it is.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusInReview  Status = "in_review"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

// Content is the editor's node tree (TipTap/ProseMirror), stored in a JSONB
// column and passed through untouched.
//
// It is deliberately not decoded into Go structs. Nothing on the server reads
// the tree except Flatten, which walks it generically — modelling every node
// type here would mean a migration of Go code every time the editor gains a
// mark.
type Content json.RawMessage

func (c Content) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "{}", nil
	}
	return string(c), nil
}

func (c *Content) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*c = Content("{}")
	case []byte:
		*c = Content(v)
	case string:
		*c = Content(v)
	default:
		return stderrors.New("documents.content: unexpected column type")
	}
	return nil
}

func (c Content) MarshalJSON() ([]byte, error) {
	if len(c) == 0 {
		return []byte("{}"), nil
	}
	return c, nil
}

func (c *Content) UnmarshalJSON(data []byte) error {
	*c = append((*c)[0:0], data...)
	return nil
}

type Document struct {
	app.BaseModel `gorm:"embedded"`
	SpaceID       uuid.UUID  `json:"space_id" gorm:"column:space_id"`
	CategoryID    *uuid.UUID `json:"category_id" gorm:"column:category_id"`
	Title         string     `json:"title" gorm:"column:title"`
	Slug          string     `json:"slug" gorm:"column:slug"`
	Content       Content    `json:"content" gorm:"column:content"`
	// Excerpt is derived from Content on every write, never supplied by the
	// client — two fields that can disagree about the same text is a bug
	// waiting for someone to edit one of them.
	Excerpt *string `json:"excerpt" gorm:"column:excerpt"`
	// SearchBody is the flattened plain text of Content, which the generated
	// search_vector column is built from — a JSON tree is not text. Derived
	// on every write beside Excerpt, never sent to a client, and never read by
	// anything but the search query and the snippet it cuts (Modul 3 FR-1).
	SearchBody *string `json:"-" gorm:"column:search_body"`
	Status     Status  `json:"status" gorm:"column:status"`
	// ProtectionLevel is nil for "inherit the space" (Modul 4 FR-6), which is
	// what the PRD's absent per-document setting row meant.
	ProtectionLevel *protection.Level `json:"protection_level" gorm:"column:protection_level"`
	// SpaceProtectionLevel is not a column on this table. GetById joins the
	// owning space for it so resolving the level costs no second query on the
	// path every document open takes. Read-only ("->"), so the full-row Save
	// never tries to write it back to a table it does not belong to.
	SpaceProtectionLevel protection.Level `json:"-" gorm:"column:space_protection_level;->"`
	// CurrentVersionID carries no foreign key: documents and document_versions
	// reference each other, which no insert order satisfies.
	CurrentVersionID *uuid.UUID `json:"current_version_id" gorm:"column:current_version_id"`
	AuthorID         uuid.UUID  `json:"author_id" gorm:"column:author_id"`
	LastEditorID     *uuid.UUID `json:"last_editor_id" gorm:"column:last_editor_id"`
	// PublishedAt is when it last went live, and is never cleared on unpublish
	// — it doubles as "has this ever been published", which is what decides
	// whether an edit is worth a version snapshot (FR-5).
	PublishedAt *time.Time     `json:"published_at" gorm:"column:published_at"`
	ViewCount   uint           `json:"view_count" gorm:"column:view_count"`
	IsPinned    bool           `json:"is_pinned" gorm:"column:is_pinned"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at"`
}

func (Document) TableName() string {
	return "documents"
}

// setContent writes the content and everything derived from it.
//
// Excerpt and SearchBody are two views of the same text, and both are stale
// the moment one of them is updated without the other — a preview that shows
// the old opening, or a search hit on words the document no longer contains.
// Every write goes through here so that cannot be half-done.
func (d *Document) setContent(content Content) {
	d.Content = content
	d.Excerpt = Excerpt(content, ExcerptLength)

	if text := PlainText(content); text != "" {
		d.SearchBody = &text
	} else {
		d.SearchBody = nil
	}
}

// HasBeenPublished decides whether an edit takes a snapshot first. A draft
// nobody has ever seen has no reader to protect (FR-5).
func (d *Document) HasBeenPublished() bool {
	return d.PublishedAt != nil
}

// EffectiveProtection is the level this document is actually served at: its
// own if it carries one, otherwise the space's (FR-6).
//
// Only meaningful on a document loaded by GetById — nothing else joins the
// space, so on a document from a list SpaceProtectionLevel is empty and this
// falls back to normal. That is the right answer for a list, which shows no
// content to protect.
func (d *Document) EffectiveProtection() protection.Level {
	return protection.Resolve(d.SpaceProtectionLevel, d.ProtectionLevel)
}

// Version is one snapshot of a document's content, taken before an edit
// overwrote it.
type Version struct {
	ID            uuid.UUID  `json:"id" gorm:"column:id;primary_key;type:uuid"`
	DocumentID    uuid.UUID  `json:"document_id" gorm:"column:document_id"`
	VersionNumber uint       `json:"version_number" gorm:"column:version_number"`
	Content       Content    `json:"content" gorm:"column:content"`
	EditorID      *uuid.UUID `json:"editor_id" gorm:"column:editor_id"`
	ChangeNote    *string    `json:"change_note" gorm:"column:change_note"`
	CreatedAt     time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (Version) TableName() string {
	return "document_versions"
}

func (v *Version) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
