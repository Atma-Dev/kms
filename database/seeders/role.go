package seeders

import (
	"api-kms/app"
	"api-kms/app/rbac"
	"api-kms/enums"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type roleSeed struct {
	ID   string
	Name enums.Role
}

var RoleSeeder = []roleSeed{
	{ID: "9b8f868a-2f75-4bb4-af0d-75a1fd5aa921", Name: enums.SUPER_ADMIN},
	{ID: "9b8f868a-2f75-4bb4-af0d-75a1fd5aa922", Name: enums.ADMIN},
	{ID: "9b8f868a-2f75-4bb4-af0d-75a1fd5aa923", Name: enums.GRAY_USER},

	// The PRD's content roles. Seeded immutable like the three above: the
	// ladder is named in the code and the docs, so renaming "Reviewer" would
	// make both lie. Their *permissions* stay editable in the role editor —
	// IsMutable only guards the name and the row's existence.
	{ID: "9b8f868a-2f75-4bb4-af0d-75a1fd5aa924", Name: enums.VIEWER},
	{ID: "9b8f868a-2f75-4bb4-af0d-75a1fd5aa925", Name: enums.CONTRIBUTOR},
	{ID: "9b8f868a-2f75-4bb4-af0d-75a1fd5aa926", Name: enums.EDITOR},
	{ID: "9b8f868a-2f75-4bb4-af0d-75a1fd5aa927", Name: enums.REVIEWER},
	{ID: "9b8f868a-2f75-4bb4-af0d-75a1fd5aa928", Name: enums.SPACE_ADMIN},
}

type Role struct{}

func (s *Role) Run(db *gorm.DB) error {
	for _, detail := range RoleSeeder {
		id, err := uuid.Parse(detail.ID)
		if err != nil {
			return err
		}

		role := rbac.Role{
			BaseModel: app.BaseModel{ID: id},
			Name:      string(detail.Name),
			IsMutable: false,
		}

		result := db.Where("name = ?", role.Name).FirstOrCreate(&role)
		if result.Error != nil {
			return result.Error
		}

		log.Printf("seeded role: %s", detail.Name)
	}

	return nil
}
