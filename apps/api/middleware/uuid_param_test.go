package middleware

import (
	"api-kms/internal/testdb"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// A malformed id must stop before it reaches Postgres, which would answer it
// with a 22P02 error — a 500 — rather than an empty result.
func TestValidateUUIDParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	run := func(params gin.Params, query string) int {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
		c.Params = params

		ValidateUUIDParams()(c)

		if len(c.Errors) == 0 {
			return 0
		}
		return testdb.HTTPStatus(c.Errors[0].Err)
	}

	cases := []struct {
		name   string
		params gin.Params
		query  string
		want   int
	}{
		{"valid path id passes", gin.Params{{Key: "id", Value: uuid.NewString()}}, "", 0},
		{"malformed path id is a 404", gin.Params{{Key: "id", Value: "not-a-uuid"}}, "", http.StatusNotFound},
		{"malformed spaceId is a 404", gin.Params{{Key: "spaceId", Value: "42"}}, "", http.StatusNotFound},
		{"token is not a uuid and is left alone", gin.Params{{Key: "token", Value: "abc"}}, "", 0},
		{"versionNumber is left alone", gin.Params{{Key: "versionNumber", Value: "3"}}, "", 0},
		{"valid filter passes", nil, "space_id=" + uuid.NewString(), 0},
		{"malformed filter is a 422", nil, "author_id=me", http.StatusUnprocessableEntity},
		{"other query params are left alone", nil, "q=hello&page=2", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := run(tc.params, tc.query); got != tc.want {
				t.Fatalf("got status %d, want %d", got, tc.want)
			}
		})
	}
}
