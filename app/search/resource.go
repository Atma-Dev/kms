package search

import (
	"api-kms/app/document"
	"api-kms/utils"
)

// ResultResource is a document as a search result: the list shape every other
// document listing uses, plus the passage that matched.
//
// Snippet is HTML and already escaped — see Snippet. The client renders it as
// markup, which is the whole point of the <mark> in it.
type ResultResource struct {
	document.DocumentResource
	Snippet string `json:"snippet"`
}

func NewResultResourceCollection(results []Result) []ResultResource {
	out := make([]ResultResource, len(results))
	for i := range results {
		out[i] = ResultResource{
			DocumentResource: document.NewDocumentResource(&results[i].Document),
			Snippet:          results[i].Snippet,
		}
	}
	return out
}

// SuggestionResource is deliberately thinner than a document: an autocomplete
// row needs a label and somewhere to go, and shipping excerpts and view counts
// for eight of them on every keystroke is the one place in the API where the
// payload size is felt.
type SuggestionResource struct {
	ID      string `json:"id"`
	SpaceID string `json:"space_id"`
	Title   string `json:"title"`
	Slug    string `json:"slug"`
}

func NewSuggestionResourceCollection(documents []document.Document) []SuggestionResource {
	out := make([]SuggestionResource, len(documents))
	for i := range documents {
		out[i] = SuggestionResource{
			ID:      documents[i].ID.String(),
			SpaceID: documents[i].SpaceID.String(),
			Title:   documents[i].Title,
			Slug:    documents[i].Slug,
		}
	}
	return out
}

type HistoryResource struct {
	ID    string `json:"id"`
	Query string `json:"query"`
	// SearchedAt is updated_at under a name that says what it means here: the
	// last time this query was run, which is what the list is ordered by.
	SearchedAt string `json:"searched_at"`
}

func NewHistoryResourceCollection(histories []History) []HistoryResource {
	out := make([]HistoryResource, len(histories))
	for i := range histories {
		out[i] = HistoryResource{
			ID:         histories[i].ID.String(),
			Query:      histories[i].Query,
			SearchedAt: utils.FormatDateTimeVal(histories[i].UpdatedAt),
		}
	}
	return out
}
