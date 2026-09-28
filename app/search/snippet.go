package search

import (
	"html"
	"strings"
	"unicode"
)

// SnippetWidth is how much of the document a result shows: enough to read the
// matched sentence, short enough that ten results still fit on a screen.
const SnippetWidth = 200

// Snippet is the PRD's "snippet dengan kata kunci di-highlight" — MySQL has no
// ts_headline, so the window and the highlighting are cut here.
//
// The returned string is HTML: every character of the document is escaped, and
// the only markup in it is the <mark> this function puts there. That order
// matters — escaping after inserting the marks would escape the marks too, and
// escaping the document only where it is not highlighted would let a document
// containing "<script>" ship it to the browser intact.
//
// Everything is done on runes rather than bytes. The match is found in a
// lowercased copy and rendered from the original, and only a rune-wise
// ToLower is guaranteed to leave the two the same length — a byte-wise one
// shifts every offset after the first non-ASCII character.
//
// A match extends to the end of its word: the index does prefix matching, so
// "deploy" finds "deployment", and highlighting only the typed half of a word
// looks like a rendering bug.
func Snippet(body string, terms []string, width int) string {
	// Whitespace is squeezed first. Documents written through the app arrive
	// normalised already, but the ones search_body was backfilled for do not —
	// and a snippet full of runs of spaces reads as a broken renderer rather
	// than as a migration artefact.
	runes := []rune(strings.Join(strings.Fields(body), " "))
	if len(runes) == 0 {
		return ""
	}

	lower := make([]rune, len(runes))
	for i, r := range runes {
		lower[i] = unicode.ToLower(r)
	}

	needles := make([][]rune, 0, len(terms))
	for _, term := range terms {
		if term == "" {
			continue
		}
		needles = append(needles, []rune(strings.ToLower(term)))
	}

	// Centre the window on the first match, with a little of what came before
	// it: a snippet that starts exactly at the keyword reads like a fragment.
	first := matchAt(lower, needles, 0)
	start := 0
	if first >= 0 {
		start = first - width/3
		if start < 0 {
			start = 0
		}
	}
	start = forwardToWordStart(runes, start)

	end := start + width
	if end > len(runes) {
		end = len(runes)
	} else {
		end = backToWordEnd(runes, end)
	}

	var b strings.Builder
	if start > 0 {
		b.WriteString("…")
	}

	for i := start; i < end; {
		length := matchLengthAt(lower, needles, i)
		if length == 0 {
			b.WriteString(html.EscapeString(string(runes[i])))
			i++
			continue
		}

		// To the end of the word, not the end of the typed prefix.
		stop := i + length
		for stop < end && (unicode.IsLetter(runes[stop]) || unicode.IsDigit(runes[stop])) {
			stop++
		}

		b.WriteString("<mark>")
		b.WriteString(html.EscapeString(string(runes[i:stop])))
		b.WriteString("</mark>")
		i = stop
	}

	if end < len(runes) {
		b.WriteString("…")
	}
	return b.String()
}

// matchAt returns the first index at or after from where any needle starts, or
// -1.
func matchAt(lower []rune, needles [][]rune, from int) int {
	for i := from; i < len(lower); i++ {
		if matchLengthAt(lower, needles, i) > 0 {
			return i
		}
	}
	return -1
}

// matchLengthAt returns the length of the longest needle starting exactly at
// i, or 0. Longest rather than first so overlapping terms ("deploy",
// "deployment") highlight as one run instead of nesting.
func matchLengthAt(lower []rune, needles [][]rune, i int) int {
	best := 0
	for _, needle := range needles {
		if len(needle) <= best || i+len(needle) > len(lower) {
			continue
		}
		if string(lower[i:i+len(needle)]) == string(needle) {
			best = len(needle)
		}
	}
	return best
}

// forwardToWordStart moves the cut forward off the middle of a word, so a
// snippet never opens on "…loyment guide".
func forwardToWordStart(runes []rune, i int) int {
	for i > 0 && i < len(runes) && !unicode.IsSpace(runes[i-1]) {
		i++
	}
	return i
}

// backToWordEnd moves the cut back off the middle of a word, so a snippet
// never closes on "deploy…" when the word was "deployment".
func backToWordEnd(runes []rune, i int) int {
	for i > 0 && i < len(runes) && !unicode.IsSpace(runes[i]) {
		i--
	}
	return i
}

// Terms is what Snippet highlights: the words the caller typed, stripped of
// the boolean operators booleanQuery removes, but *not* filtered by
// the index's minimum token size — a two-letter word the index could not
// match may still have
// been found by the LIKE fallback, and leaving it unhighlighted would make the
// result look like a mistake.
func Terms(raw string) []string {
	terms := make([]string, 0, 8)
	for _, field := range strings.Fields(raw) {
		term := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, field)
		if term != "" {
			terms = append(terms, term)
		}
	}
	return terms
}
