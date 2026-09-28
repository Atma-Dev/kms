package middleware

import (
	"api-kms/app/m2mclient"
	"api-kms/config"
	"api-kms/internal/testdb"
	"api-kms/utils"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Covers the branches that decide whether a machine-to-machine caller gets in:
// right secret, wrong secret, revoked client, and no credentials at all.
func TestM2MAuthMiddleware(t *testing.T) {
	db := testdb.Open(t)
	config.DB = db

	gin.SetMode(gin.TestMode)
	plaintextSecret := "correct-secret"
	hashed, err := utils.MakeHash(plaintextSecret)
	if err != nil {
		t.Fatalf("MakeHash failed: %v", err)
	}

	// Client is just a display name now — X-Client-Id is matched against the
	// row's id (uuid), not this string, so it is set here but never used as a
	// credential.
	live := m2mclient.M2MClient{Client: "TEST-" + uuid.NewString()[:8], Secret: hashed}
	if err := db.Create(&live).Error; err != nil {
		t.Fatalf("failed to seed live client: %v", err)
	}
	defer db.Unscoped().Delete(&live)

	revokedAt := time.Now()
	revoked := m2mclient.M2MClient{Client: "TEST-" + uuid.NewString()[:8], Secret: hashed, RevokedAt: &revokedAt}
	if err := db.Create(&revoked).Error; err != nil {
		t.Fatalf("failed to seed revoked client: %v", err)
	}
	defer db.Unscoped().Delete(&revoked)

	run := func(setCredentials func(*http.Request)) (*gin.Context, int) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if setCredentials != nil {
			setCredentials(req)
		}
		c.Request = req

		M2MAuthMiddleware()(c)

		if len(c.Errors) == 0 {
			return c, 0
		}
		return c, testdb.HTTPStatus(c.Errors[0].Err)
	}

	setHeaders := func(id, secret string) func(*http.Request) {
		return func(r *http.Request) {
			r.Header.Set("X-Client-Id", id)
			r.Header.Set("X-Client-Secret", secret)
		}
	}

	t.Run("valid credentials pass through", func(t *testing.T) {
		c, status := run(setHeaders(live.ID.String(), plaintextSecret))
		if status != 0 {
			t.Fatalf("got status %d, want no error", status)
		}
		got, exists := c.Get("m2m_client")
		if !exists {
			t.Fatal("m2m_client was not set in context")
		}
		if got.(*m2mclient.M2MClient).ID != live.ID {
			t.Fatalf("m2m_client.ID = %s, want %s", got.(*m2mclient.M2MClient).ID, live.ID)
		}
	})

	t.Run("wrong secret rejected", func(t *testing.T) {
		_, status := run(setHeaders(live.ID.String(), "wrong-secret"))
		if status != 401 {
			t.Fatalf("got status %d, want 401", status)
		}
	})

	t.Run("revoked client rejected", func(t *testing.T) {
		_, status := run(setHeaders(revoked.ID.String(), plaintextSecret))
		if status != 401 {
			t.Fatalf("got status %d, want 401", status)
		}
	})

	t.Run("client name is not a valid credential", func(t *testing.T) {
		_, status := run(setHeaders(live.Client, plaintextSecret))
		if status != 401 {
			t.Fatalf("got status %d, want 401 — the display name must not authenticate", status)
		}
	})

	t.Run("missing credentials rejected", func(t *testing.T) {
		_, status := run(nil)
		if status != 401 {
			t.Fatalf("got status %d, want 401", status)
		}
	})
}
