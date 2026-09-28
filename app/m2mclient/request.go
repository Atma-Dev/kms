package m2mclient

// Uniqueness on client is enforced by the unique index, not by the
// unique_column validator, which has no way to exclude the row being updated.
type StoreM2MClientRequest struct {
	Client string   `form:"client" json:"client" binding:"required,max=255"`
	Scope  []string `form:"scope" json:"scope" binding:"required,min=1,dive,oneof=*"`
}

type UpdateM2MClientRequest struct {
	Client string   `form:"client" json:"client" binding:"required,max=255"`
	Scope  []string `form:"scope" json:"scope" binding:"required,min=1,dive,oneof=*"`
}
