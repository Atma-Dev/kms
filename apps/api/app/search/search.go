// Package search is Modul 3: finding a document by what is written in it,
// rather than by clicking down to it through spaces and categories.
//
// # The index
//
// What the PRD specifies: documents.search_vector, a tsvector generated from
// the title (weight A) and search_body (weight C), behind a GIN index, ranked
// with ts_rank. Postgres generates the column, so every write keeps it in step
// and no code path can forget to.
//
// # The index is not a service
//
// The NFR asks that the index backend be replaceable without changing the
// /search contract. That seam is the HTTP response, not a Go interface: there
// is exactly one implementation, and an interface with one implementation
// hides which code runs without making anything easier to replace. Swapping in
// Meilisearch means rewriting Repository.Search; every caller and every client
// sees the same JSON either way.
//
// # What a caller may find
//
// Search answers with the same set the space list would: restricted documents
// (FR-9) the caller is not named on never appear, and only published documents
// are searchable — except the caller's own drafts, which they wrote and are
// entitled to find again.
package search

import (
	"api-kms/app"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// TitleWeight is the PRD's setweight('A') for the title against setweight('C')
// for the body. A document whose *title* is the query is almost always the one
// that was meant; a document that mentions it once in a paragraph rarely is.
const TitleWeight = 3

// rankWeights is the weight array ts_rank takes, in its {D, C, B, A} order:
// the body is C and the title A, TitleWeight times as heavy. Spelled out
// rather than left to ts_rank's default {0.1, 0.2, 0.4, 1.0}, whose title is
// worth five bodies — a number nobody here chose.
var rankWeights = fmt.Sprintf("{0, %g, 0, %g}", 0.1, 0.1*TitleWeight)

// History is one remembered query (FR-5), one row per user per distinct query
// — see the migration for why that is a unique key and not an append log.
type History struct {
	app.BaseModel `gorm:"embedded"`
	UserID        uuid.UUID `json:"user_id" gorm:"column:user_id"`
	Query         string    `json:"query" gorm:"column:query"`
}

func (History) TableName() string {
	return "search_histories"
}

// tsQuery turns what someone typed into a to_tsquery expression.
//
// Three things happen here, and all are load-bearing:
//
// Operators are stripped. In to_tsquery `&`, `|`, `!`, `<->`, `:`, `*`, `'`
// and parentheses are syntax, so a query like "C++ (draft)" is either a
// syntax error or a search for something the user did not ask for. Nobody
// types those meaning to drive the parser. Terms keeps only letters and
// digits, so each term is safe inside its quotes.
//
// Every term gets `:*`. Prefix matching is what makes a search box feel like
// it is keeping up: "deploy" should find "deployment" while it is still being
// typed.
//
// Terms are OR-ed with `|` rather than AND-ed. Requiring all of them means a
// four-word question finds nothing; OR-ing them and letting ts_rank sort it
// out means the document matching all four comes first anyway.
//
// An empty result means nothing typed was a word at all — the caller falls
// back to an ILIKE.
func tsQuery(raw string) string {
	terms := Terms(raw)
	for i, term := range terms {
		terms[i] = "'" + strings.ToLower(term) + "':*"
	}
	return strings.Join(terms, " | ")
}
