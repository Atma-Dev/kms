package user

import (
	"api-kms/app/rbac"
	"slices"
	"testing"
)

// A user with two roles sharing a permission must see it once in the
// flattened list, not once per role.
func TestNewResourcePermissionsDeduped(t *testing.T) {
	u := &User{
		Roles: []rbac.Role{
			{Permissions: []rbac.Permission{{Name: "deposits.show"}, {Name: "deposits.update"}}},
			{Permissions: []rbac.Permission{{Name: "deposits.show"}, {Name: "payments.void"}}},
		},
	}

	got := NewResource(u).Permissions

	want := []string{"deposits.show", "deposits.update", "payments.void"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, name := range want {
		if !slices.Contains(got, name) {
			t.Fatalf("got %v, missing %q", got, name)
		}
	}
}
