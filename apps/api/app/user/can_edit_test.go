package user

import (
	"api-kms/app"
	"api-kms/app/rbac"
	"api-kms/enums"
	"testing"

	"github.com/google/uuid"
)

func withPermissions(id uuid.UUID, roleName string, names ...enums.Permission) *User {
	role := rbac.Role{BaseModel: app.BaseModel{ID: uuid.New()}, Name: roleName}
	for _, name := range names {
		role.Permissions = append(role.Permissions, rbac.Permission{
			BaseModel: app.BaseModel{ID: uuid.New()}, Name: string(name),
		})
	}
	return &User{BaseModel: app.BaseModel{ID: id}, Roles: []rbac.Role{role}}
}

// The PRD's Contributor/Editor split. A route gate cannot see whose document
// it is, so this is the rule the service applies to the row — and the one that
// decides whether someone can rewrite a colleague's work.
func TestCanEditDocumentSeparatesContributorFromEditor(t *testing.T) {
	mine, theirs := uuid.New(), uuid.New()

	contributor := withPermissions(mine, "Contributor", enums.PermissionDocumentsUpdate)
	if !contributor.CanEditDocument(mine.String()) {
		t.Error("a contributor cannot edit their own document")
	}
	if contributor.CanEditDocument(theirs.String()) {
		t.Error("a contributor can edit someone else's document")
	}

	// Holding .all implies the narrower grant, so a role only ever needs the
	// one that matches what it is.
	editor := withPermissions(mine, "Editor", enums.PermissionDocumentsUpdateAll)
	if !editor.CanEditDocument(theirs.String()) || !editor.CanEditDocument(mine.String()) {
		t.Error("an editor should be able to edit anyone's document")
	}
}

func TestCanEditDocumentRefusesSomeoneWithNeitherGrant(t *testing.T) {
	id := uuid.New()
	reader := withPermissions(id, "Reader", enums.PermissionDocumentsShow)

	if reader.CanEditDocument(id.String()) {
		t.Error("documents.show alone should not allow editing, even of one's own document")
	}
}

// Super-Admin short-circuits every check. If that ever stopped being true the
// whole app would lock its own administrator out, silently.
func TestSuperAdminHoldsEveryPermission(t *testing.T) {
	admin := &User{
		BaseModel: app.BaseModel{ID: uuid.New()},
		Roles:     []rbac.Role{{BaseModel: app.BaseModel{ID: uuid.New()}, Name: string(enums.SUPER_ADMIN)}},
	}

	if !admin.HasPermission(enums.PermissionDocumentsAccessManage) {
		t.Error("Super-Admin was refused a permission it holds no explicit row for")
	}
	if !admin.CanEditDocument(uuid.New().String()) {
		t.Error("Super-Admin cannot edit someone else's document")
	}
}

// A user with no roles at all must not fall through to "allowed".
func TestUserWithNoRolesHoldsNothing(t *testing.T) {
	nobody := &User{BaseModel: app.BaseModel{ID: uuid.New()}}

	if nobody.HasPermission(enums.PermissionDocumentsShow) {
		t.Error("a user with no roles was granted a permission")
	}
	if nobody.CanEditDocument(nobody.ID.String()) {
		t.Error("a user with no roles could edit their own document")
	}
}
