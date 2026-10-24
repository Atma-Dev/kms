package search

import (
	"api-kms/app/document"
	"api-kms/utils"
	"context"
)

// Engine is the index behind /search.
//
// The NFR asks that the index backend be replaceable without changing the
// /search contract, and this is the seam it is replaceable at: Query carries
// no HTTP and no gorm, and the results are documents the service turns into
// the same JSON either way. Moving to Meilisearch or Elasticsearch means a
// second implementation of these two methods and one line in NewContainer.
//
// It deliberately does **not** cover the search history. That is ordinary rows
// in this application's own database — it would stay exactly where it is if
// the index moved to another process tomorrow, and putting it behind this
// interface would oblige every future engine to reimplement a table it has
// nothing to do with.
type Engine interface {
	// Search returns one page of hits, ordered by Query.Sort, with the meta
	// the client pages through.
	Search(ctx context.Context, q Query) ([]document.Document, *utils.PaginationMeta, error)

	// Suggest returns titles beginning with Query.Text — the autocomplete,
	// which is a prefix match rather than a search (FR-6).
	Suggest(ctx context.Context, q Query) ([]document.Document, error)
}

// Repository is the MySQL implementation, and the only one today.
var _ Engine = (*Repository)(nil)
