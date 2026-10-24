package category

import "github.com/google/uuid"

// Slug is derived from the name, not accepted from the client — same reason as
// app/space: two categories must not be able to disagree about a URL.
type StoreCategoryRequest struct {
	Name     string     `form:"name" json:"name" binding:"required,max=255"`
	ParentID *uuid.UUID `form:"parent_id" json:"parent_id"`
}

// UpdateCategoryRequest is a full replace, as PUT implies: an absent parent_id
// means "a root category", not "leave it where it was".
//
// A partial-update shape was tried and dropped — encoding/json cannot tell an
// absent parent_id from an explicit null (it sets the pointer to nil either
// way), so "move to root" and "don't move" would arrive identical.
type UpdateCategoryRequest struct {
	Name     string     `form:"name" json:"name" binding:"required,max=255"`
	ParentID *uuid.UUID `form:"parent_id" json:"parent_id"`
}

// ReorderCategoriesRequest takes the whole ordered sibling list, not a single
// moved id: positions are renumbered 0..n-1 in one transaction, and a
// half-renumbered list leaves two categories sharing a position.
type ReorderCategoriesRequest struct {
	ParentID *uuid.UUID  `form:"parent_id" json:"parent_id"`
	IDs      []uuid.UUID `form:"ids" json:"ids" binding:"required,min=1"`
}
