package search

import (
	"api-kms/errors"
	"api-kms/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Query is everything an Engine needs to answer one search, with nothing in it
// that knows about HTTP.
//
// It exists so the interface in engine.go can be implemented by something that
// is not reading a *gin.Context — which is the whole point of the interface,
// and would be a lie if the engine still had to be handed the request.
type Query struct {
	// Text is what the user typed, trimmed. Empty means "browse the filters",
	// which is a legitimate search: the filters alone are a useful list.
	Text string

	SpaceID    string
	CategoryID string
	TagID      string
	AuthorID   string
	// From and To bound documents.updated_at. Nil means unbounded on that
	// side; To is inclusive of its whole day — see ParseQuery.
	From *time.Time
	To   *time.Time

	// Sort is "newest", "popular", or anything else for relevance. Validated
	// nowhere on purpose: an unknown value falls back to the default rather
	// than erroring, so a stale bookmark still renders a list.
	Sort string

	Page    int
	PerPage int
	// Path is what PaginationMeta echoes back to the client. A response
	// concern rather than a search one, carried here because it is the only
	// thing the pager still needs from the request.
	Path string

	// ViewerID decides which drafts are searchable — the caller's own.
	ViewerID uuid.UUID
	// Hidden is the restricted documents (FR-9) this caller is not named on,
	// resolved by app/document. An engine applies it; it never works it out.
	Hidden []uuid.UUID
}

// Sort values. Relevance is the zero value: it is what an unrecognised sort
// falls back to, and what an empty one means.
const (
	SortRelevance = ""
	SortNewest    = "newest"
	SortPopular   = "popular"
)

// ParseQuery reads one search request. The only place in this package that
// touches a *gin.Context.
func ParseQuery(c *gin.Context, viewerID uuid.UUID, hidden []uuid.UUID) (Query, error) {
	page, perPage, err := utils.PageParams(c)
	if err != nil {
		return Query{}, err
	}

	from, err := parseDate(c.Query("from"), "from")
	if err != nil {
		return Query{}, err
	}
	to, err := parseDate(c.Query("to"), "to")
	if err != nil {
		return Query{}, err
	}
	if to != nil {
		// The end of that day, not its midnight: a range ending "today" that
		// excluded everything edited today is the reliably wrong answer.
		end := to.AddDate(0, 0, 1)
		to = &end
	}

	path := ""
	if c.Request != nil {
		path = c.Request.URL.Path
	}

	return Query{
		Text:       strings.TrimSpace(c.Query("q")),
		SpaceID:    c.Query("space_id"),
		CategoryID: c.Query("category_id"),
		TagID:      c.Query("tag"),
		AuthorID:   c.Query("author_id"),
		From:       from,
		To:         to,
		Sort:       c.Query("sort"),
		Page:       page,
		PerPage:    perPage,
		Path:       path,
		ViewerID:   viewerID,
		Hidden:     hidden,
	}, nil
}

func parseDate(value, field string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, errors.InvalidStatusException("date filters must look like 2006-01-02").
			WithErrors(map[string][]string{field: {"must look like 2006-01-02"}}).
			WithCause(err)
	}
	return &parsed, nil
}
