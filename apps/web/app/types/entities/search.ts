import type Document from './document'

/**
 * A search hit: the document list row, plus the passage that matched.
 *
 * `snippet` is HTML — the API escapes the document's own text and wraps the
 * matched words in `<mark>`, so it is rendered with `v-html` rather than
 * interpolated. Highlighting here instead would mean a second implementation
 * of "what matched", and the client does not know how the query was tokenised.
 */
export interface SearchResult extends Document {
  snippet: string
}

/** One autocomplete row: a label and somewhere to go, nothing else. */
export interface SearchSuggestion {
  id: string
  space_id: string
  title: string
  slug: string
}

export interface SearchHistoryEntry {
  id: string
  query: string
  searched_at: string
}

export type SearchSort = 'relevance' | 'newest' | 'popular'
