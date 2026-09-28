package middleware

import (
	"api-kms/app/m2mclient"
	"api-kms/internal/testdb"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Covers the branches that decide whether an already-authenticated M2M
// client may call a specific route: exact scope match, "*" wildcard, and no
// matching scope at all.
func TestRequireM2MScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	run := func(scopes m2mclient.Scopes, required string) (*gin.Context, int) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set("m2m_client", &m2mclient.M2MClient{Scope: scopes})

		RequireM2MScope(required)(c)

		if len(c.Errors) == 0 {
			return c, 0
		}
		return c, testdb.HTTPStatus(c.Errors[0].Err)
	}

	t.Run("exact scope match passes", func(t *testing.T) {
		_, status := run(m2mclient.Scopes{"invoices"}, "invoices")
		if status != 0 {
			t.Fatalf("got status %d, want no error", status)
		}
	})

	t.Run("wildcard grants any scope", func(t *testing.T) {
		_, status := run(m2mclient.Scopes{m2mclient.ScopeAll}, "memberships")
		if status != 0 {
			t.Fatalf("got status %d, want no error", status)
		}
	})

	t.Run("missing scope rejected", func(t *testing.T) {
		_, status := run(m2mclient.Scopes{"invoices"}, "memberships")
		if status != 403 {
			t.Fatalf("got status %d, want 403", status)
		}
	})
}
