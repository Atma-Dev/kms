// Package space owns the top-level container everything else in the content
// module hangs off: one space per team, product or domain.
//
// A space is archived, never deleted. It holds documents, and removing the row
// would either orphan them or cascade the delete into content someone wrote —
// archiving takes it out of the pickers and leaves its documents readable.
package space

import (
	"api-kms/app"
	"api-kms/app/protection"

	"github.com/google/uuid"
)

type Space struct {
	app.BaseModel `gorm:"embedded"`
	Name          string  `json:"name" gorm:"column:name"`
	Slug          string  `json:"slug" gorm:"column:slug"`
	Description   *string `json:"description" gorm:"column:description"`
	Icon          *string `json:"icon" gorm:"column:icon"`
	IsArchived    bool    `json:"is_archived" gorm:"column:is_archived"`
	// ProtectionLevel is the default every document in the space inherits
	// unless it carries one of its own (Modul 4 FR-6).
	ProtectionLevel protection.Level `json:"protection_level" gorm:"column:protection_level"`
	// CreatedBy is who opened the space. Kept separate from the audit trail:
	// that answers "who changed it", this is shown on the space itself.
	CreatedBy uuid.UUID `json:"created_by" gorm:"column:created_by"`
}

func (Space) TableName() string {
	return "spaces"
}
