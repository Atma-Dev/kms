package lookup

// Option is one entry of a select dropdown. Deliberately the smallest thing
// that can fill one — every consumer maps its source down to exactly this, so
// a dropdown never ships the full resource just to read two fields.
type Option struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
