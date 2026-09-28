package tag

type StoreTagRequest struct {
	Name string `form:"name" json:"name" binding:"required,max=64"`
}

// SyncDocumentTagsRequest replaces a document's whole tag set, rather than
// adding or removing one at a time. The editor holds the final set, and a
// per-tag endpoint would need the client to diff it — which is a second
// implementation of "what changed" waiting to disagree with the first.
//
// `names` rather than ids: a tag the user just typed has no id yet, and making
// them create it first would be two round trips for one action.
type SyncDocumentTagsRequest struct {
	Names []string `form:"names" json:"names" binding:"omitempty,dive,max=64"`
}

type UpdateTagRequest struct {
	Name string `form:"name" json:"name" binding:"required,max=64"`
}
