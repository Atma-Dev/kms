package document

import (
	"api-kms/app/protection"
	"api-kms/app/tag"
	"api-kms/app/user"
	"api-kms/utils"
)

type DocumentResource struct {
	ID         string  `json:"id"`
	SpaceID    string  `json:"space_id"`
	CategoryID *string `json:"category_id"`
	Title      string  `json:"title"`
	Slug       string  `json:"slug"`
	Excerpt    *string `json:"excerpt"`
	Status     Status  `json:"status"`
	AuthorID   string  `json:"author_id"`
	ViewCount  uint    `json:"view_count"`
	IsPinned   bool    `json:"is_pinned"`

	PublishedAt *string `json:"published_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// DocumentDetailResource is the single-document response. Content is here and
// not on the list resource — a page of twenty documents would otherwise ship
// twenty full node trees to render twenty titles.
type DocumentDetailResource struct {
	DocumentResource
	Content          Content `json:"content"`
	LastEditorID     *string `json:"last_editor_id"`
	CurrentVersionID *string `json:"current_version_id"`

	// AvailableActions is what this caller may do to this document right now —
	// the legal transitions from its status, filtered by what they hold. The
	// client renders exactly these buttons instead of re-deriving the rules
	// and drifting from them.
	AvailableActions []Action `json:"available_actions"`
	CanEdit          bool     `json:"can_edit"`

	Tags []tag.TagResource `json:"tags"`
	// IsRestricted says the document carries access grants, so the reader is
	// looking at it because they are named on it. The client shows a lock; it
	// never has to work out who is on the list, which is gated separately.
	IsRestricted bool `json:"is_restricted"`

	// ProtectionLevel is the level actually in force (Modul 4 FR-6), space
	// default already applied — it is what the viewer renders from, so the
	// client is never left to resolve the inheritance itself.
	ProtectionLevel protection.Level `json:"protection_level"`
	// ProtectionLevelOverride is the document's own setting, null when it
	// inherits. The two are separate because the settings form has to show
	// "inherit" as a distinct choice from "normal", which the resolved value
	// cannot express.
	ProtectionLevelOverride *protection.Level `json:"protection_level_override"`
	// ContentHidden says the text was withheld because the document is strict
	// and this caller may only read it. The client then loads the page images
	// instead — without this flag an empty content tree would be
	// indistinguishable from an empty document.
	ContentHidden bool `json:"content_hidden"`
}

func NewDocumentResource(d *Document) DocumentResource {
	var categoryID *string
	if d.CategoryID != nil {
		id := d.CategoryID.String()
		categoryID = &id
	}

	return DocumentResource{
		ID:          d.ID.String(),
		SpaceID:     d.SpaceID.String(),
		CategoryID:  categoryID,
		Title:       d.Title,
		Slug:        d.Slug,
		Excerpt:     d.Excerpt,
		Status:      d.Status,
		AuthorID:    d.AuthorID.String(),
		ViewCount:   d.ViewCount,
		IsPinned:    d.IsPinned,
		PublishedAt: utils.FormatDateTime(d.PublishedAt),
		CreatedAt:   utils.FormatDateTimeVal(d.CreatedAt),
		UpdatedAt:   utils.FormatDateTimeVal(d.UpdatedAt),
	}
}

func NewDocumentResourceCollection(documents []Document) []DocumentResource {
	result := make([]DocumentResource, len(documents))
	for i := range documents {
		result[i] = NewDocumentResource(&documents[i])
	}
	return result
}

func NewDocumentDetailResource(d *Document, actor *user.User, tags []tag.Tag, isRestricted, contentVisible bool) DocumentDetailResource {
	var lastEditorID, currentVersionID *string
	if d.LastEditorID != nil {
		id := d.LastEditorID.String()
		lastEditorID = &id
	}
	if d.CurrentVersionID != nil {
		id := d.CurrentVersionID.String()
		currentVersionID = &id
	}

	actions := make([]Action, 0, len(Available(d.Status)))
	for _, action := range Available(d.Status) {
		if CanTransition(d, action, actor) {
			actions = append(actions, action)
		}
	}

	// Withheld rather than emptied at the source: the document keeps its
	// content, and only this one response omits it. Asking the browser not to
	// let anyone select text it was sent anyway would be a lock beside an open
	// window — at the strict level the text simply does not leave the server.
	content := d.Content
	if !contentVisible {
		content = Content("{}")
	}

	return DocumentDetailResource{
		DocumentResource:        NewDocumentResource(d),
		Content:                 content,
		LastEditorID:            lastEditorID,
		CurrentVersionID:        currentVersionID,
		AvailableActions:        actions,
		CanEdit:                 CanEdit(d, actor),
		Tags:                    tag.NewTagResourceCollection(tags),
		IsRestricted:            isRestricted,
		ProtectionLevel:         d.EffectiveProtection(),
		ProtectionLevelOverride: d.ProtectionLevel,
		ContentHidden:           !contentVisible,
	}
}

// VersionResource is the version list: numbers, who and why, no content. The
// list is a sidebar — the content is fetched one version at a time.
type VersionResource struct {
	ID            string  `json:"id"`
	VersionNumber uint    `json:"version_number"`
	EditorID      *string `json:"editor_id"`
	ChangeNote    *string `json:"change_note"`
	CreatedAt     string  `json:"created_at"`
}

// VersionDetailResource adds the snapshot itself, for viewing one version.
type VersionDetailResource struct {
	VersionResource
	Content Content `json:"content"`
}

func NewVersionResource(v *Version) VersionResource {
	var editorID *string
	if v.EditorID != nil {
		id := v.EditorID.String()
		editorID = &id
	}

	return VersionResource{
		ID:            v.ID.String(),
		VersionNumber: v.VersionNumber,
		EditorID:      editorID,
		ChangeNote:    v.ChangeNote,
		CreatedAt:     utils.FormatDateTimeVal(v.CreatedAt),
	}
}

func NewVersionResourceCollection(versions []Version) []VersionResource {
	result := make([]VersionResource, len(versions))
	for i := range versions {
		result[i] = NewVersionResource(&versions[i])
	}
	return result
}

func NewVersionDetailResource(v *Version) VersionDetailResource {
	return VersionDetailResource{
		VersionResource: NewVersionResource(v),
		Content:         v.Content,
	}
}

// AccessResource is one grant as the access panel shows it.
type AccessResource struct {
	ID          string      `json:"id"`
	SubjectType SubjectType `json:"subject_type"`
	SubjectID   string      `json:"subject_id"`
	Permission  AccessLevel `json:"permission"`
	CreatedAt   string      `json:"created_at"`
}

func NewAccessResource(a *Access) AccessResource {
	return AccessResource{
		ID:          a.ID.String(),
		SubjectType: a.SubjectType,
		SubjectID:   a.SubjectID.String(),
		Permission:  a.Permission,
		CreatedAt:   utils.FormatDateTimeVal(a.CreatedAt),
	}
}

func NewAccessResourceCollection(grants []Access) []AccessResource {
	result := make([]AccessResource, len(grants))
	for i := range grants {
		result[i] = NewAccessResource(&grants[i])
	}
	return result
}
