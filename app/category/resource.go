package category

import "api-kms/utils"

type CategoryResource struct {
	ID       string             `json:"id"`
	SpaceID  string             `json:"space_id"`
	ParentID *string            `json:"parent_id"`
	Name     string             `json:"name"`
	Slug     string             `json:"slug"`
	Order    int                `json:"order"`
	Children []CategoryResource `json:"children"`
	// DocumentCount includes everything in the subtree, because that is what
	// clicking this category lists. Zero on the plain create/update responses,
	// which do not count — only the tree endpoint fills it.
	DocumentCount int64 `json:"document_count"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func NewCategoryResource(c *Category) CategoryResource {
	var parentID *string
	if c.ParentID != nil {
		id := c.ParentID.String()
		parentID = &id
	}

	return CategoryResource{
		ID:       c.ID.String(),
		SpaceID:  c.SpaceID.String(),
		ParentID: parentID,
		Name:     c.Name,
		Slug:     c.Slug,
		Order:    c.Order,
		// Never nil: a leaf serialises as [] rather than null, so the client
		// can recurse without a type check at every level.
		Children:  []CategoryResource{},
		CreatedAt: utils.FormatDateTimeVal(c.CreatedAt),
		UpdatedAt: utils.FormatDateTimeVal(c.UpdatedAt),
	}
}

// NewCategoryTreeResource renders the nested shape BuildTree produced.
func NewCategoryTreeResource(nodes []Node) []CategoryResource {
	result := make([]CategoryResource, len(nodes))
	for i := range nodes {
		resource := NewCategoryResource(&nodes[i].Category)
		resource.Children = NewCategoryTreeResource(nodes[i].Children)
		resource.DocumentCount = nodes[i].DocumentCount
		result[i] = resource
	}
	return result
}

// CategoryTreeResponse wraps the tree with the two numbers that have no node
// of their own: everything in the space, and everything filed nowhere.
type CategoryTreeResponse struct {
	Categories []CategoryResource `json:"categories"`
	// TotalDocumentCount backs the "all documents" row above the tree.
	TotalDocumentCount int64 `json:"total_document_count"`
	// UncategorisedCount is documents with no category. They are reachable
	// through "all documents" and nowhere else, so the number is worth showing
	// rather than leaving them to be discovered by subtraction.
	UncategorisedCount int64 `json:"uncategorised_count"`
}
