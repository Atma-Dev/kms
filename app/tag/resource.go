package tag

import "api-kms/utils"

type TagResource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	// DocumentCount is only filled by the master list — the tag chips on a
	// document have no use for it, and counting on every read would cost a
	// join the document page never needs.
	DocumentCount *int64 `json:"document_count,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func NewTagResource(t *Tag) TagResource {
	return TagResource{
		ID:        t.ID.String(),
		Name:      t.Name,
		Slug:      t.Slug,
		CreatedAt: utils.FormatDateTimeVal(t.CreatedAt),
		UpdatedAt: utils.FormatDateTimeVal(t.UpdatedAt),
	}
}

func NewTagResourceCollection(tags []Tag) []TagResource {
	result := make([]TagResource, len(tags))
	for i := range tags {
		result[i] = NewTagResource(&tags[i])
	}
	return result
}

// WithCounts is the master list's shape: the same rows plus how many documents
// each one labels, so deleting a tag shows what it would take with it.
func NewTagResourceCollectionWithCounts(tags []Tag, counts map[string]int64) []TagResource {
	result := make([]TagResource, len(tags))
	for i := range tags {
		result[i] = NewTagResource(&tags[i])
		count := counts[tags[i].ID.String()]
		result[i].DocumentCount = &count
	}
	return result
}
