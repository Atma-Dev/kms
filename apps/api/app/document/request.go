package document

import "github.com/google/uuid"

// ExcerptLength is how much of the content is kept as the list preview.
const ExcerptLength = 300

// Slug is derived from the title, never accepted from the client — same rule
// as app/space and app/category.
//
// Content is the editor's node tree, passed through untouched. It is validated
// as "some JSON object" and no further: the server does not model node types
// (see the Content doc comment).
type StoreDocumentRequest struct {
	Title      string     `form:"title" json:"title" binding:"required,max=255"`
	CategoryID *uuid.UUID `form:"category_id" json:"category_id"`
	Content    Content    `form:"content" json:"content" binding:"required"`
}

// UpdateDocumentRequest is a full replace, as PUT implies: an absent
// category_id files the document at the space root.
//
// ChangeNote is stored on the snapshot this edit produces, not on the
// document, so it describes the change rather than the current state.
type UpdateDocumentRequest struct {
	Title      string     `form:"title" json:"title" binding:"required,max=255"`
	CategoryID *uuid.UUID `form:"category_id" json:"category_id"`
	Content    Content    `form:"content" json:"content" binding:"required"`
	ChangeNote *string    `form:"change_note" json:"change_note" binding:"omitempty,max=500"`
	IsPinned   *bool      `form:"is_pinned" json:"is_pinned"`
}

// TransitionRequest is the body every lifecycle endpoint takes. Only reject
// requires a reason — the others accept a note and mostly get none.
type TransitionRequest struct {
	Reason *string `form:"reason" json:"reason" binding:"omitempty,max=500"`
}

// RollbackRequest optionally records why the document was rolled back. The
// rollback itself snapshots the content it replaces, so nothing is lost either
// way.
type RollbackRequest struct {
	ChangeNote *string `form:"change_note" json:"change_note" binding:"omitempty,max=500"`
}

// StoreAccessRequest restricts a document to one subject (FR-9). The first
// grant on a document narrows it; it never widens one — see the Access doc
// comment.
type StoreAccessRequest struct {
	SubjectType SubjectType `form:"subject_type" json:"subject_type" binding:"required,oneof=user role"`
	SubjectID   uuid.UUID   `form:"subject_id" json:"subject_id" binding:"required"`
	Permission  AccessLevel `form:"permission" json:"permission" binding:"required,oneof=view edit manage"`
}

// PinDocumentRequest carries only the flag. Pinning deliberately does not go
// through PUT: sending the whole document to flip one boolean would carry the
// caller's copy of the content with it and overwrite anything edited in
// between.
type PinDocumentRequest struct {
	IsPinned *bool `form:"is_pinned" json:"is_pinned" binding:"required"`
}
