package seeders

import (
	"api-kms/app/rbac"
	"api-kms/enums"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Permission makes the permissions table match enums.PermissionSeeder exactly:
// rows added, rows updated, and rows no longer in the enum removed.
//
// The removal is the part that used to be missing, and it mattered more than it
// looks. Nothing here ever deleted, so a permission renamed or dropped in code
// stayed in the table forever — and RolePermission grants Super-Admin every row
// it finds in the table rather than every entry in the enum, so the leftover
// kept being granted. A gate deleted from the code went on existing as an
// unreferenced capability nobody could see in a diff.
//
// The enum is the source of truth: a permission that no longer appears there
// gates nothing, so keeping the row only leaves rubbish for whoever next reads
// the permission screen. role_has_permissions is cleaned up by its own
// ON DELETE CASCADE.
type Permission struct{}

func (s *Permission) Run(db *gorm.DB) error {
	names := make([]string, 0, len(enums.PermissionSeeder))

	for _, detail := range enums.PermissionSeeder {
		permission := rbac.Permission{
			Name:         string(detail.Name),
			Description:  detail.Description,
			FeatureGroup: detail.FeatureGroup,
		}

		result := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"description", "feature_group"}),
		}).Create(&permission)

		if result.Error != nil {
			return result.Error
		}

		names = append(names, permission.Name)
	}

	var stale []rbac.Permission
	if err := db.Where("name NOT IN ?", names).Find(&stale).Error; err != nil {
		return err
	}
	for _, permission := range stale {
		// One line each rather than a count: a permission disappearing is a
		// capability disappearing, and it should be readable in the deploy log
		// which one it was.
		log.Printf("removed rbac permission no longer in the enum: %s", permission.Name)
	}
	if len(stale) > 0 {
		if err := db.Delete(&stale).Error; err != nil {
			return err
		}
	}

	log.Printf("seeded rbac: %d permissions, %d removed", len(names), len(stale))
	return nil
}
