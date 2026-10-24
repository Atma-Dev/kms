package activitylog

import (
	"context"
	"api-kms/internal/testdb"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// listContext fakes the request GetAllPaginated reads its filters off.
func listContext(query string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/management/audits?"+query, nil)
	return c
}

func TestCreate(t *testing.T) {
	db := testdb.Open(t)
	repo := NewRepository(db)

	userID := "test-user"
	log := &ActivityLog{Method: "GET", Path: "/api/test", StatusCode: 200, IP: "127.0.0.1", UserID: &userID}
	if err := repo.Create(context.Background(), log); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if log.ID.String() == "" {
		t.Fatal("expected ID to be generated")
	}

	var found ActivityLog
	if err := db.First(&found, "id = ?", log.ID).Error; err != nil {
		t.Fatalf("failed to load created row: %v", err)
	}
	if found.Path != "/api/test" || found.StatusCode != 200 {
		t.Fatalf("unexpected row: %+v", found)
	}
}

func TestGetAllPaginatedFilters(t *testing.T) {
	db := testdb.Open(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.NewString()
	target := &ActivityLog{Method: "POST", Path: "/api/filter-target", StatusCode: 201, IP: "127.0.0.1", UserID: &userID}
	other := &ActivityLog{Method: "GET", Path: "/api/other", StatusCode: 500, IP: "127.0.0.1"}
	if err := repo.Create(ctx, target); err != nil {
		t.Fatalf("failed to seed target row: %v", err)
	}
	if err := repo.Create(ctx, other); err != nil {
		t.Fatalf("failed to seed other row: %v", err)
	}
	t.Cleanup(func() {
		db.Unscoped().Delete(target)
		db.Unscoped().Delete(other)
	})

	cases := []struct {
		name  string
		query string
	}{
		{"search", "search=filter-target"},
		{"method", "method=post"},
		{"status_code", "status_code=201"},
		{"user_id", "user_id=" + userID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logs, _, err := repo.GetAllPaginated(listContext(tc.query))
			if err != nil {
				t.Fatalf("GetAllPaginated(%q) failed: %v", tc.query, err)
			}
			var sawTarget, sawOther bool
			for _, l := range logs {
				if l.ID == target.ID {
					sawTarget = true
				}
				if l.ID == other.ID {
					sawOther = true
				}
			}
			if !sawTarget {
				t.Fatalf("query %q excluded the row it should have matched", tc.query)
			}
			if sawOther {
				t.Fatalf("query %q included a row it should have excluded", tc.query)
			}
		})
	}
}
