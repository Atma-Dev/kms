package seeders

import (
	"api-kms/app/rbac"
	"api-kms/enums"
	"api-kms/internal/testdb"
	"testing"
)

// TestPermissionRemovesStale is the check on the part of the sync that deletes.
//
// Adding is easy to notice — a missing permission shows up as a 403 the first
// time someone hits the new endpoint. Failing to remove is silent: the row sits
// in the table, RolePermission keeps granting it because it grants whatever the
// table holds, and nothing in the app ever mentions it again.
func TestPermissionRemovesStale(t *testing.T) {
	db := testdb.Open(t)

	ghost := rbac.Permission{
		Name:         "test.ghost.show",
		Description:  "a permission the enum does not have",
		FeatureGroup: "test",
	}
	if err := db.Create(&ghost).Error; err != nil {
		t.Fatalf("could not insert the stale permission: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&rbac.Permission{}, "name = ?", ghost.Name) })

	if err := (&Permission{}).Run(db); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var stillThere int64
	if err := db.Model(&rbac.Permission{}).Where("name = ?", ghost.Name).Count(&stillThere).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if stillThere != 0 {
		t.Fatal("a permission missing from the enum survived the sync")
	}

	// And the enum's own entries are all present afterwards, so "remove what is
	// not in the enum" cannot pass by emptying the table.
	var kept int64
	if err := db.Model(&rbac.Permission{}).Count(&kept).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if kept != int64(len(enums.PermissionSeeder)) {
		t.Fatalf("permissions table holds %d rows, the enum has %d", kept, len(enums.PermissionSeeder))
	}
}
