package middleware

import (
	"api-kms/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// uuidParams are the route parameters that name a UUID primary key. :token
// and :versionNumber are deliberately absent — neither is one.
var uuidParams = []string{"id", "spaceId", "sessionId", "accessId"}

// uuidFilters are the query-string filters that compare against a UUID
// column wherever they appear. Search's ?tag= is one too, but the name is too
// generic to claim for every route, so app/search checks it itself.
var uuidFilters = []string{"space_id", "category_id", "author_id", "user_id"}

// ValidateUUIDParams rejects a malformed id before any handler runs. Postgres
// answers a comparison between a uuid column and garbage text with an error
// (22P02), not an empty result — so without this gate a mistyped id turns
// into a 500.
//
// A path id answers 404, which is what it means: there is no such resource.
// A filter answers 422 naming the field, like the date filters do — a filter
// that silently matched nothing would read as "no results".
func ValidateUUIDParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, name := range uuidParams {
			if value := c.Param(name); value != "" {
				if _, err := uuid.Parse(value); err != nil {
					abortWithError(c, errors.EmptyDataException("resource not found").WithCause(err))
					return
				}
			}
		}
		for _, name := range uuidFilters {
			if value := c.Query(name); value != "" {
				if _, err := uuid.Parse(value); err != nil {
					abortWithError(c, errors.InvalidStatusException(name+" must be a UUID").
						WithErrors(map[string][]string{name: {"must be a UUID"}}).
						WithCause(err))
					return
				}
			}
		}
	}
}
