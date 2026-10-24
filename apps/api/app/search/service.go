package search

import (
	"api-kms/app/document"
	"api-kms/app/user"
	"api-kms/utils"
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

// maxRememberedQuery is the history column's width. A query longer than this
// is searched normally and simply not remembered — truncating it would offer
// the user a saved search that is not the one they ran.
const maxRememberedQuery = 255

// Result is one hit: the document, plus the piece of it that matched.
type Result struct {
	Document document.Document
	Snippet  string
}

type Service struct {
	// engine is the index. An interface, so the backend behind /search can be
	// replaced without this file changing — see engine.go.
	engine Engine
	// history is the search history store, which is not part of Engine: those
	// are this application's own rows and would not move with the index.
	history   *Repository
	documents *document.Service
}

// Search answers FR-1 through FR-3 and remembers the query (FR-5).
//
// The hidden set is resolved by app/document, and narrowed to the space being
// filtered on when there is one: the answer is the same either way, and asking
// for one space's restricted documents beats asking for every space's.
func (s *Service) Search(c *gin.Context, viewer *user.User) ([]Result, *utils.PaginationMeta, error) {
	ctx := c.Request.Context()

	hidden, err := s.documents.HiddenFrom(ctx, c.Query("space_id"), viewer)
	if err != nil {
		return nil, nil, err
	}

	query, err := ParseQuery(c, viewer.ID, hidden)
	if err != nil {
		return nil, nil, err
	}

	documents, meta, err := s.engine.Search(ctx, query)
	if err != nil {
		return nil, nil, err
	}

	// Best effort, like the view count: nobody should be shown an error page
	// because their search history did not save.
	if query.Text != "" && len(query.Text) <= maxRememberedQuery {
		_ = s.history.RememberQuery(ctx, viewer.ID, query.Text)
	}

	terms := Terms(query.Text)
	results := make([]Result, len(documents))
	for i := range documents {
		results[i] = Result{
			Document: documents[i],
			Snippet:  Snippet(snippetSource(&documents[i]), terms, SnippetWidth),
		}
	}
	return results, meta, nil
}

// snippetSource prefers the indexed body over the excerpt: the excerpt is only
// the opening, and a match further down the document would be highlighted
// nowhere if the snippet could only ever be cut from the first 500 characters.
func snippetSource(doc *document.Document) string {
	if doc.SearchBody != nil {
		return *doc.SearchBody
	}
	if doc.Excerpt != nil {
		return *doc.Excerpt
	}
	return ""
}

func (s *Service) Suggest(c *gin.Context, viewer *user.User) ([]document.Document, error) {
	ctx := c.Request.Context()

	if strings.TrimSpace(c.Query("q")) == "" {
		// Nothing has been typed, so nothing is being completed. An empty list
		// rather than the most-viewed documents: a dropdown that appears
		// before the first keystroke is in the way, not helpful.
		return []document.Document{}, nil
	}

	hidden, err := s.documents.HiddenFrom(ctx, c.Query("space_id"), viewer)
	if err != nil {
		return nil, err
	}

	query, err := ParseQuery(c, viewer.ID, hidden)
	if err != nil {
		return nil, err
	}
	return s.engine.Suggest(ctx, query)
}

func (s *Service) Histories(ctx context.Context, viewer *user.User) ([]History, error) {
	return s.history.Histories(ctx, viewer.ID)
}

func (s *Service) DeleteHistory(ctx context.Context, viewer *user.User, id string) error {
	return s.history.DeleteHistory(ctx, viewer.ID, id)
}

// NewService takes the engine and the history store separately even though one
// *Repository is both today — that is the point of the split, and passing the
// same value twice is what it looks like while Postgres is the only engine.
func NewService(engine Engine, history *Repository, documents *document.Service) *Service {
	return &Service{engine: engine, history: history, documents: documents}
}
