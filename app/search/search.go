// Package search is Modul 3: finding a document by what is written in it,
// rather than by clicking down to it through spaces and categories.
//
// # Why this is not a tsvector
//
// The PRD specifies Postgres — tsvector, setweight, ts_rank, a GIN index.
// This codebase is MySQL, so the equivalent is InnoDB FULLTEXT: two indexes
// (documents.title and documents.search_body) matched separately, and the
// PRD's A/B/C weighting applied as arithmetic at query time —
// MATCH(title) * TitleWeight + MATCH(search_body). MySQL scores one index at
// a time, which is what makes the weight a multiplication here instead of a
// property of the index.
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
	"strings"

	"github.com/google/uuid"
)

// TitleWeight is the PRD's setweight('A') for the title against setweight('C')
// for the body. A document whose *title* is the query is almost always the one
// that was meant; a document that mentions it once in a paragraph rarely is.
const TitleWeight = 3

// DefaultMinTokenSize is InnoDB's own default for innodb_ft_min_token_size.
// It is only the fallback: NewRepository reads the running server's value,
// because an install that tuned it down would otherwise keep taking the LIKE
// fallback for words the index now holds — and one that tuned it up would
// search the index for words that are not in it.
//
// Words shorter than the limit are not indexed at all, which is what decides
// whether a query can use the index or has to fall back.
const DefaultMinTokenSize = 3

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

// booleanQuery turns what someone typed into a MySQL BOOLEAN MODE expression.
//
// Two things happen here, and both are load-bearing:
//
// Operators are stripped. In boolean mode `+`, `-`, `*`, `"`, `(`, `)`, `~`,
// `<` and `>` are syntax, so a query like "C++ (draft)" is either a syntax
// error or a search for something the user did not ask for. Nobody types those
// meaning to drive the parser.
//
// Every surviving term gets a trailing `*`. Prefix matching is what makes a
// search box feel like it is keeping up: "deploy" should find "deployment"
// while it is still being typed.
//
// Terms are left implicitly OR-ed rather than each being required with `+`.
// Requiring all of them means a four-word question finds nothing; OR-ing them
// and letting the relevance score sort it out means the document matching all
// four comes first anyway.
//
// An empty result means nothing here is indexable at minToken — the caller
// falls back to a LIKE.
func booleanQuery(raw string, minToken int) string {
	terms := make([]string, 0, 8)
	for _, term := range Terms(raw) {
		if len([]rune(term)) < minToken {
			continue
		}
		terms = append(terms, term+"*")
	}
	return strings.Join(terms, " ")
}
