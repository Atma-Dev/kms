package space

// Uniqueness on slug is enforced by the unique index, not by a validator —
// a validator has no way to exclude the row being updated.
//
// Slug is not accepted from the client: it is derived from the name (see
// utils.Slugify) so two spaces cannot disagree about which one owns a URL.
type StoreSpaceRequest struct {
	Name        string  `form:"name" json:"name" binding:"required,max=255"`
	Description *string `form:"description" json:"description" binding:"omitempty,max=2000"`
	Icon        *string `form:"icon" json:"icon" binding:"omitempty,max=64"`
}

type UpdateSpaceRequest struct {
	Name        string  `form:"name" json:"name" binding:"required,max=255"`
	Description *string `form:"description" json:"description" binding:"omitempty,max=2000"`
	Icon        *string `form:"icon" json:"icon" binding:"omitempty,max=64"`
	IsArchived  *bool   `form:"is_archived" json:"is_archived" binding:"omitempty"`
}
