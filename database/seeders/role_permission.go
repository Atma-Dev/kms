package seeders

import (
	"api-kms/app/rbac"
	"api-kms/enums"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

// RolePermission grants every seeded permission to Super-Admin, so a fresh dev
// database doesn't need each new gate assigned by hand before it can be
// exercised. Re-running is safe: it replaces Super-Admin's permission set with
// "all of them" each time, so a newly added permission is picked up on the next
// `db:seed` without needing a migration of its own.
//
// The PRD's content roles get a starting set too, but only while they hold
// nothing at all. Every one of them is unusable empty — a Contributor without
// documents.store cannot open a single form — so shipping them blank would
// just mean everyone repeats the same clicks in the role editor.
//
// Anything beyond that opening set, and any change to it, is a real decision
// that belongs to whoever runs the deployment, so this never overwrites a role
// someone has already configured.
//
// Admin and Gray-User are deliberately absent: what they mean varies per
// deployment, and unlike the ladder below there is no answer that is obviously
// right.
type RolePermission struct{}

// browse is what every role needs before it can see anything at all. Reading
// the space list and the category tree is gated by masters.*.show despite the
// name — "masters" is where the resource lives, not how privileged it is.
var browse = []enums.Permission{
	enums.PermissionDocumentsShow,
	enums.PermissionMastersSpacesShow,
	enums.PermissionMastersCategoriesShow,
	enums.PermissionLookupsSpacesShow,
	enums.PermissionLookupsCategoriesShow,
	// Without this the tag field still works, but silently stops suggesting
	// the tags that already exist — which is how one vocabulary becomes three.
	enums.PermissionLookupsTagsShow,
}

// Each rung extends the one above it, so the ladder is visible here rather
// than hidden in five hand-copied lists that drift apart on the first edit.
var (
	viewerGrants = browse

	contributorGrants = extend(viewerGrants,
		enums.PermissionDocumentsStore,
		// The narrower grant: edits what they authored. See
		// user.User.CanEditDocument for the rule this turns into.
		enums.PermissionDocumentsUpdate,
	)

	// Not given documents.update as well: holding .all already implies it.
	editorGrants = extend(viewerGrants,
		enums.PermissionDocumentsStore,
		enums.PermissionDocumentsUpdateAll,
	)

	// approve / reject / publish / unpublish, which the PRD grants together.
	reviewerGrants = extend(editorGrants, enums.PermissionDocumentsReview)

	spaceAdminGrants = extend(reviewerGrants,
		enums.PermissionDocumentsArchive,
		enums.PermissionDocumentsDestroy,
		enums.PermissionDocumentsAccessManage,

		// Modul 4. All three sit here rather than lower down the ladder: the
		// protection level and the share link both decide how far a document
		// travels, and the access log is a record of what people read. The PRD
		// puts every one of them behind "Admin".
		enums.PermissionDocumentsProtectionManage,
		enums.PermissionDocumentsShare,
		enums.PermissionDocumentsAccessLogShow,

		// Restricting a document names users and roles, so the picker needs
		// both lookups — the grant is useless without them.
		enums.PermissionLookupsUsersShow,
		enums.PermissionLookupsRolesShow,

		enums.PermissionMastersSpacesStore,
		enums.PermissionMastersSpacesUpdate,
		enums.PermissionMastersSpacesDestroy,

		enums.PermissionMastersCategoriesStore,
		enums.PermissionMastersCategoriesUpdate,
		enums.PermissionMastersCategoriesDestroy,

		// Curating the vocabulary: renaming a tag, and deleting one out of
		// every document that uses it.
		enums.PermissionMastersTagsShow,
		enums.PermissionMastersTagsStore,
		enums.PermissionMastersTagsUpdate,
		enums.PermissionMastersTagsDestroy,
	)
)

// extend copies rather than appending in place: append would share the base
// slice's array, so the second rung built from the same base would overwrite
// the first one's tail.
func extend(base []enums.Permission, extra ...enums.Permission) []enums.Permission {
	out := make([]enums.Permission, 0, len(base)+len(extra))
	out = append(out, base...)
	return append(out, extra...)
}

// defaultGrants is the smallest set that makes each role do its job at all.
var defaultGrants = map[enums.Role][]enums.Permission{
	enums.VIEWER:      viewerGrants,
	enums.CONTRIBUTOR: contributorGrants,
	enums.EDITOR:      editorGrants,
	enums.REVIEWER:    reviewerGrants,
	enums.SPACE_ADMIN: spaceAdminGrants,
}

func (s *RolePermission) Run(db *gorm.DB) error {
	var superAdmin rbac.Role
	if err := db.Where("name = ?", string(enums.SUPER_ADMIN)).First(&superAdmin).Error; err != nil {
		return err
	}

	var permissions []rbac.Permission
	if err := db.Find(&permissions).Error; err != nil {
		return err
	}

	if err := db.Model(&superAdmin).Association("Permissions").Replace(permissions); err != nil {
		return err
	}
	log.Printf("seeded rbac: %s granted all %d permissions", enums.SUPER_ADMIN, len(permissions))

	for _, roleName := range ladder {
		if err := seedDefaultGrants(db, roleName, defaultGrants[roleName]); err != nil {
			return err
		}
	}

	return nil
}

// ladder fixes the order the grants are seeded in. Ranging the map directly
// would log them in a different order on every run, which turns a diff of two
// seed logs into noise.
var ladder = []enums.Role{
	enums.VIEWER, enums.CONTRIBUTOR, enums.EDITOR, enums.REVIEWER, enums.SPACE_ADMIN,
}

// seedDefaultGrants fills a role that holds nothing, and leaves one that has
// already been configured exactly as it is.
// missingFromDefaults lists the permissions a configured role's default set
// names but the role does not hold. Order follows the default set, so a
// permission added at the end of it reads last here too.
func missingFromDefaults(db *gorm.DB, role *rbac.Role, grants []enums.Permission) ([]string, error) {
	var current []rbac.Permission
	if err := db.Model(role).Association("Permissions").Find(&current); err != nil {
		return nil, err
	}

	held := make(map[string]bool, len(current))
	for _, permission := range current {
		held[permission.Name] = true
	}

	var missing []string
	for _, grant := range grants {
		if !held[string(grant)] {
			missing = append(missing, string(grant))
		}
	}
	return missing, nil
}

func seedDefaultGrants(db *gorm.DB, roleName enums.Role, grants []enums.Permission) error {
	var role rbac.Role
	if err := db.Where("name = ?", string(roleName)).First(&role).Error; err != nil {
		return err
	}

	if held := db.Model(&role).Association("Permissions").Count(); held > 0 {
		// Leaving it untouched is right — someone configured this role, and a
		// seeder must not overwrite that. But saying only "left untouched"
		// hides the case that actually bites: a permission added to the
		// default set after the role was first filled never reaches it, and
		// nothing ever says so. Naming the gap turns a silent divergence into
		// a line an operator can act on in the role editor.
		missing, err := missingFromDefaults(db, &role, grants)
		if err != nil {
			return err
		}
		if len(missing) > 0 {
			log.Printf("seeded rbac: %s already has %d permissions, left untouched — missing from its default set: %s",
				roleName, held, strings.Join(missing, ", "))
			return nil
		}

		log.Printf("seeded rbac: %s already has %d permissions, left untouched", roleName, held)
		return nil
	}

	names := make([]string, len(grants))
	for i, g := range grants {
		names[i] = string(g)
	}

	var permissions []rbac.Permission
	if err := db.Where("name IN ?", names).Find(&permissions).Error; err != nil {
		return err
	}
	if len(permissions) != len(names) {
		// A grant naming a permission that no longer exists would silently
		// hand the role a smaller set than intended, and nothing downstream
		// would ever say so.
		return fmt.Errorf("seeder for %s names %d permissions but only %d exist", roleName, len(names), len(permissions))
	}

	if err := db.Model(&role).Association("Permissions").Replace(permissions); err != nil {
		return err
	}

	log.Printf("seeded rbac: %s granted %d starting permissions", roleName, len(permissions))
	return nil
}
