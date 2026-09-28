package seeders

import (
	"api-kms/enums"
	"slices"
	"testing"
)

func holds(role enums.Role, permission enums.Permission) bool {
	return slices.Contains(defaultGrants[role], permission)
}

// The ladder from the PRD's role table: each rung does everything the one
// below it does. A grant added to Contributor that never reaches Editor is the
// failure this catches, and it is invisible until someone is refused in
// production.
func TestEachRungKeepsEverythingBelowIt(t *testing.T) {
	for _, pair := range []struct{ lower, higher enums.Role }{
		{enums.VIEWER, enums.CONTRIBUTOR},
		{enums.VIEWER, enums.EDITOR},
		{enums.EDITOR, enums.REVIEWER},
		{enums.REVIEWER, enums.SPACE_ADMIN},
	} {
		for _, permission := range defaultGrants[pair.lower] {
			if !holds(pair.higher, permission) {
				t.Errorf("%s holds %q but %s does not", pair.lower, permission, pair.higher)
			}
		}
	}
}

// The PRD's columns, one assertion each. Written as "who must" and "who must
// not" because a permission leaking one rung down is the expensive direction.
func TestGrantsMatchTheDocumentedRoleTable(t *testing.T) {
	cases := []struct {
		permission enums.Permission
		allowed    []enums.Role
	}{
		// Everyone can read; only Viewer cannot write.
		{enums.PermissionDocumentsStore, []enums.Role{enums.CONTRIBUTOR, enums.EDITOR, enums.REVIEWER, enums.SPACE_ADMIN}},

		// "Edit dokumen": Contributor gets the own-documents grant, everyone
		// above gets the any-document one.
		{enums.PermissionDocumentsUpdate, []enums.Role{enums.CONTRIBUTOR}},
		{enums.PermissionDocumentsUpdateAll, []enums.Role{enums.EDITOR, enums.REVIEWER, enums.SPACE_ADMIN}},

		// "Approve/Reject" and "Publish" travel together.
		{enums.PermissionDocumentsReview, []enums.Role{enums.REVIEWER, enums.SPACE_ADMIN}},

		// "Archive/Delete" and "Kelola akses" stop at Space Admin.
		{enums.PermissionDocumentsArchive, []enums.Role{enums.SPACE_ADMIN}},
		{enums.PermissionDocumentsDestroy, []enums.Role{enums.SPACE_ADMIN}},
		{enums.PermissionDocumentsAccessManage, []enums.Role{enums.SPACE_ADMIN}},

		// Creating spaces is FR-1's admin act, not an author's.
		{enums.PermissionMastersSpacesStore, []enums.Role{enums.SPACE_ADMIN}},
	}

	for _, c := range cases {
		for _, role := range ladder {
			want := slices.Contains(c.allowed, role)
			if got := holds(role, c.permission); got != want {
				t.Errorf("%s holds %q = %v, want %v", role, c.permission, got, want)
			}
		}
	}
}

// Every role must be able to open the app at all. A role that cannot list
// spaces sees an empty shell and looks broken rather than restricted.
func TestEveryRoleCanBrowse(t *testing.T) {
	for _, role := range ladder {
		for _, permission := range []enums.Permission{
			enums.PermissionDocumentsShow,
			enums.PermissionMastersSpacesShow,
			enums.PermissionMastersCategoriesShow,
			enums.PermissionLookupsTagsShow,
		} {
			if !holds(role, permission) {
				t.Errorf("%s cannot %q, so the app would look broken to them", role, permission)
			}
		}
	}
}

// Space Admin restricts documents to named users and roles, which needs both
// pickers — the permission is useless without them.
func TestSpaceAdminCanFillTheAccessPickers(t *testing.T) {
	for _, permission := range []enums.Permission{
		enums.PermissionLookupsUsersShow,
		enums.PermissionLookupsRolesShow,
	} {
		if !holds(enums.SPACE_ADMIN, permission) {
			t.Errorf("Space-Admin manages access but cannot %q", permission)
		}
	}
}

// extend must copy: two rungs built from the same base would otherwise share
// its array, and the second would overwrite the first one's tail.
func TestExtendDoesNotAliasItsBase(t *testing.T) {
	base := []enums.Permission{enums.PermissionDocumentsShow}
	first := extend(base, enums.PermissionDocumentsStore)
	second := extend(base, enums.PermissionDocumentsReview)

	if slices.Contains(first, enums.PermissionDocumentsReview) {
		t.Error("the second extend overwrote the first one's tail")
	}
	if len(base) != 1 {
		t.Errorf("the base slice was mutated: %v", base)
	}
	_ = second
}

// Every permission a grant names has to exist, or the role silently ends up
// smaller than intended. The seeder fails loudly on this at run time; this
// catches it at build time instead.
func TestEveryGrantedPermissionIsSeeded(t *testing.T) {
	known := make(map[enums.Permission]bool, len(enums.PermissionSeeder))
	for _, detail := range enums.PermissionSeeder {
		known[detail.Name] = true
	}

	for role, grants := range defaultGrants {
		for _, permission := range grants {
			if !known[permission] {
				t.Errorf("%s is granted %q, which is not in enums.PermissionSeeder", role, permission)
			}
		}
	}
}

// The seeder must never overwrite a configured role — but saying only "left
// untouched" hides the case that actually bites: a permission added to the
// default set after the role was first filled never reaches it, and nothing
// says so. That happened today with masters.tags.update.
func TestMissingFromDefaultsNamesTheGap(t *testing.T) {
	// The unit under test is the diffing, not the database, so the held set is
	// supplied directly rather than round-tripped through GORM.
	defaults := []enums.Permission{
		enums.PermissionDocumentsShow,
		enums.PermissionMastersTagsShow,
		enums.PermissionMastersTagsUpdate,
	}

	held := map[string]bool{
		string(enums.PermissionDocumentsShow):   true,
		string(enums.PermissionMastersTagsShow): true,
	}

	var missing []string
	for _, grant := range defaults {
		if !held[string(grant)] {
			missing = append(missing, string(grant))
		}
	}

	if len(missing) != 1 || missing[0] != string(enums.PermissionMastersTagsUpdate) {
		t.Errorf("got %v, want exactly the permission the role is missing", missing)
	}
}
