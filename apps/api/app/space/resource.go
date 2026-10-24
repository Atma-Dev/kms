package space

import (
	"api-kms/app/protection"
	"api-kms/utils"
)

type SpaceResource struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	IsArchived  bool    `json:"is_archived"`
	// ProtectionLevel is the default every document in this space is served
	// at unless it carries an override of its own (Modul 4 FR-6).
	ProtectionLevel protection.Level `json:"protection_level"`
	CreatedBy       string           `json:"created_by"`
	CreatedAt       string           `json:"created_at"`
	UpdatedAt       string           `json:"updated_at"`
}

func NewSpaceResource(s *Space) SpaceResource {
	return SpaceResource{
		ID:              s.ID.String(),
		Name:            s.Name,
		Slug:            s.Slug,
		Description:     s.Description,
		Icon:            s.Icon,
		IsArchived:      s.IsArchived,
		ProtectionLevel: s.ProtectionLevel,
		CreatedBy:       s.CreatedBy.String(),
		CreatedAt:       utils.FormatDateTimeVal(s.CreatedAt),
		UpdatedAt:       utils.FormatDateTimeVal(s.UpdatedAt),
	}
}

func NewSpaceResourceCollection(spaces []Space) []SpaceResource {
	result := make([]SpaceResource, len(spaces))
	for i := range spaces {
		result[i] = NewSpaceResource(&spaces[i])
	}
	return result
}
