package middleware

import (
	"api-kms/app/rbac"
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/internal/testdb"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// §4.5d: only a user holding the given permission through any role gets in;
// everyone else (including one with unrelated permissions, or none at all)
// is rejected with 403.
func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	run := func(authUser *user.User) (*gin.Context, int) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPatch, "/", nil)
		c.Set("user", authUser)

		RequirePermission(enums.PermissionManagementRolesUpdate)(c)

		if len(c.Errors) == 0 {
			return c, 0
		}
		return c, testdb.HTTPStatus(c.Errors[0].Err)
	}

	t.Run("holder passes through", func(t *testing.T) {
		holder := &user.User{Roles: []rbac.Role{
			{Permissions: []rbac.Permission{{Name: string(enums.PermissionManagementRolesUpdate)}}},
		}}
		if _, status := run(holder); status != 0 {
			t.Fatalf("got status %d, want no error", status)
		}
	})

	t.Run("user with an unrelated permission is rejected", func(t *testing.T) {
		other := &user.User{Roles: []rbac.Role{
			{Permissions: []rbac.Permission{{Name: string(enums.PermissionManagementRolesDestroy)}}},
		}}
		if _, status := run(other); status != 403 {
			t.Fatalf("got status %d, want 403", status)
		}
	})

	t.Run("user with no roles is rejected", func(t *testing.T) {
		if _, status := run(&user.User{}); status != 403 {
			t.Fatalf("got status %d, want 403", status)
		}
	})
}

// The filter gate only fires when the parameter is actually present: a caller
// without the lookup permission must still be able to load the page, just not
// narrow it. And a present parameter is refused rather than ignored, so the
// caller never mistakes an unfiltered list for a filtered one.
func TestRequirePermissionForQueryParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	run := func(authUser *user.User, query string) int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/"+query, nil)
		c.Set("user", authUser)

		RequirePermissionForQueryParam("user_id", enums.PermissionLookupsUsersShow)(c)

		if len(c.Errors) == 0 {
			return 0
		}
		return testdb.HTTPStatus(c.Errors[0].Err)
	}

	holder := &user.User{Roles: []rbac.Role{
		{Permissions: []rbac.Permission{{Name: string(enums.PermissionLookupsUsersShow)}}},
	}}
	// Can open the page (holds management.activity-logs.show) but cannot use
	// the user filter — the case that made /activity-logs unreachable before.
	auditsOnly := &user.User{Roles: []rbac.Role{
		{Permissions: []rbac.Permission{{Name: string(enums.PermissionManagementActivityLogsShow)}}},
	}}

	t.Run("no parameter passes through without the permission", func(t *testing.T) {
		if status := run(auditsOnly, ""); status != 0 {
			t.Fatalf("got status %d, want no error", status)
		}
	})

	t.Run("empty parameter is treated as absent", func(t *testing.T) {
		if status := run(auditsOnly, "?user_id="); status != 0 {
			t.Fatalf("got status %d, want no error", status)
		}
	})

	t.Run("parameter without the permission is refused", func(t *testing.T) {
		if status := run(auditsOnly, "?user_id=abc"); status != 403 {
			t.Fatalf("got status %d, want 403", status)
		}
	})

	t.Run("holder may filter", func(t *testing.T) {
		if status := run(holder, "?user_id=abc"); status != 0 {
			t.Fatalf("got status %d, want no error", status)
		}
	})

	t.Run("an unrelated parameter does not trigger the gate", func(t *testing.T) {
		if status := run(auditsOnly, "?method=POST"); status != 0 {
			t.Fatalf("got status %d, want no error", status)
		}
	})
}

// The narrow and wide forms of the same grant: documents.update covers what I
// authored, documents.update.all covers anyone's. A route gated on the narrow
// one alone locks out every role holding only the wide one — which can do
// strictly more. That is invisible in the seeder's grant lists and only showed
// up when a seeded Editor was refused editing their own document.
func TestRequireAnyPermissionAcceptsEitherForm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	run := func(held enums.Permission) int {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPut, "/", nil)
		c.Set("user", &user.User{Roles: []rbac.Role{
			{Permissions: []rbac.Permission{{Name: string(held)}}},
		}})

		RequireAnyPermission(enums.PermissionDocumentsUpdate, enums.PermissionDocumentsUpdateAll)(c)

		if len(c.Errors) == 0 {
			return 0
		}
		return testdb.HTTPStatus(c.Errors[0].Err)
	}

	for _, held := range []enums.Permission{
		enums.PermissionDocumentsUpdate,
		enums.PermissionDocumentsUpdateAll,
	} {
		if status := run(held); status != 0 {
			t.Errorf("holding %q was refused with %d", held, status)
		}
	}

	if status := run(enums.PermissionDocumentsShow); status != 403 {
		t.Errorf("a caller holding neither grant got %d, want 403", status)
	}
}
