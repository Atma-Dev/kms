package utils

import (
	"api-kms/enums"
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	MaximumPerPage = 100
	DefaultPerPage = 10
)

// Paginate applies offset pagination to db and fills out with the current page.
// Every filter already chained on db is included in the total count.
//
// The request is read here and nowhere else; the paging itself is PaginateAt,
// which callers holding a page number rather than a *gin.Context use directly.
func Paginate[T any](c *gin.Context, db *gorm.DB, out *[]T) (*PaginationMeta, error) {
	page, perPage, err := PageParams(c)
	if err != nil {
		return nil, err
	}

	// WithContext ties both queries to the request lifetime and gives each of
	// them its own statement, so Offset/Limit never leak into the caller's db.
	if c.Request != nil {
		db = db.WithContext(c.Request.Context())
	}

	path := ""
	if c.Request != nil {
		path = c.Request.URL.Path
	}

	return PaginateAt(db, page, perPage, path, out)
}

// PageParams reads and validates ?page and ?per_page.
//
// Split out so a caller that parses the rest of the query into a value of its
// own — app/search's Query — reads these two the same way the pager does,
// instead of a second Atoi with its own idea of what is valid.
func PageParams(c *gin.Context) (page, perPage int, err error) {
	page, err = positiveIntQuery(c, "page", 1)
	if err != nil {
		return 0, 0, err
	}

	perPage, err = positiveIntQuery(c, "per_page", DefaultPerPage)
	if err != nil {
		return 0, 0, err
	}
	if perPage > MaximumPerPage {
		return 0, 0, invalidQueryParam("per_page", "may not be greater than "+strconv.Itoa(MaximumPerPage))
	}
	return page, perPage, nil
}

// PaginateAt is Paginate with the page already decided — the whole of the
// paging, none of the request reading.
func PaginateAt[T any](db *gorm.DB, page, perPage int, path string, out *[]T) (*PaginationMeta, error) {
	var total int64
	if err := db.Model(out).Count(&total).Error; err != nil {
		return nil, queryFailed(err)
	}

	lastPage := 1
	if total > 0 {
		lastPage = int(math.Ceil(float64(total) / float64(perPage)))
	}

	// Guarding on lastPage also keeps offset from overflowing on absurd page values.
	offset := 0
	if page <= lastPage && total > 0 {
		offset = (page - 1) * perPage
		if err := db.Offset(offset).Limit(perPage).Find(out).Error; err != nil {
			return nil, queryFailed(err)
		}
	} else {
		*out = []T{}
	}

	from, to := 0, 0
	if len(*out) > 0 {
		from = offset + 1
		to = offset + len(*out)
	}

	return &PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		From:        from,
		To:          to,
		LastPage:    lastPage,
		Path:        path,
		Total:       total,
	}, nil
}

// positiveIntQuery reads key as a positive integer, falling back to def when the
// parameter is absent. A present but unusable value is rejected instead of being
// silently replaced, so the client can tell why the page it asked for is missing.
func positiveIntQuery(c *gin.Context, key string, def int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return def, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, invalidQueryParam(key, "must be a positive integer")
	}
	return value, nil
}

// queryFailed lives here rather than in api-kms/errors because that package
// imports utils, so utils cannot import it back.
func queryFailed(err error) *HTTPError {
	return NewHttpError(
		"Failed to query data",
		enums.ERR_INTERNAL_SERVER_ERROR,
		nil,
	).WithCause(err)
}

func invalidQueryParam(param string, reason string) *HTTPError {
	return NewHttpError(
		"Invalid query parameter",
		enums.ERR_INVALID_QUERY_PARAMETER,
		&map[string][]string{param: {"The " + param + " " + reason + "."}},
	)
}
