package middleware

import (
	"api-kms/app/user"
	"api-kms/config"
	"api-kms/internal/testdb"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Covers the branch that closes the gap a stateless JWT otherwise leaves
// open: sso can revoke a membership after a token was already issued, and
// that has to take effect on the very next request, not just the next login.
func TestLoadTokenSubjectRejectsInactiveMembership(t *testing.T) {
	db := testdb.Open(t)
	config.DB = db
	gin.SetMode(gin.TestMode)

	number := uint64(90000001)
	active := user.User{
		FirstName:        "Active",
		Email:            uuid.NewString() + "@test.local",
		Number:           &number,
		MembershipStatus: user.MembershipStatusActive,
	}
	if err := db.Create(&active).Error; err != nil {
		t.Fatalf("failed to seed active user: %v", err)
	}
	defer db.Unscoped().Delete(&active)

	inactiveNumber := uint64(90000002)
	inactive := user.User{
		FirstName:        "Inactive",
		Email:            uuid.NewString() + "@test.local",
		Number:           &inactiveNumber,
		MembershipStatus: user.MembershipStatusInactive,
	}
	if err := db.Create(&inactive).Error; err != nil {
		t.Fatalf("failed to seed inactive user: %v", err)
	}
	defer db.Unscoped().Delete(&inactive)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	if _, httpErr := loadTokenSubject(c, active.ID.String()); httpErr != nil {
		t.Fatalf("active membership was rejected: %v", httpErr)
	}

	if _, httpErr := loadTokenSubject(c, inactive.ID.String()); httpErr == nil {
		t.Fatal("expected an inactive membership to be rejected, got no error")
	} else if httpErr.StatusCode != 401 {
		t.Fatalf("got status %d, want 401", httpErr.StatusCode)
	}
}
