package search

import "testing"

// Snippet is the one place a document's own text is turned back into HTML, so
// it is the one place a document containing markup could become markup.
func TestSnippetEscapesTheDocumentButNotItsOwnMarks(t *testing.T) {
	got := Snippet(`run <script>alert(1)</script> before deploy`, []string{"deploy"}, 200)

	want := `run &lt;script&gt;alert(1)&lt;/script&gt; before <mark>deploy</mark>`
	if got != want {
		t.Errorf("Snippet() = %q, want %q", got, want)
	}
}

// The index matches prefixes, so highlighting only the typed prefix would
// leave half a word marked and look like a rendering fault.
func TestSnippetHighlightsToTheEndOfTheWord(t *testing.T) {
	got := Snippet("the deployment pipeline", []string{"deploy"}, 200)

	if want := "the <mark>deployment</mark> pipeline"; got != want {
		t.Errorf("Snippet() = %q, want %q", got, want)
	}
}

// A match far into a long document has to pull the window with it, or the
// snippet shows an opening paragraph with nothing highlighted in it.
func TestSnippetCentresOnTheMatchAndEllipsisesBothEnds(t *testing.T) {
	body := ""
	for range 40 {
		body += "filler words here "
	}
	body += "the migration ran " + body

	got := Snippet(body, []string{"migration"}, 60)

	if len([]rune(got)) > 120 {
		t.Errorf("snippet is %d runes, far past the 60 asked for: %q", len([]rune(got)), got)
	}
	if !contains(got, "<mark>migration</mark>") {
		t.Errorf("the match is not in the window: %q", got)
	}
	if got[:3] != "…" && string([]rune(got)[0]) != "…" {
		t.Errorf("a window cut from the middle should open with an ellipsis: %q", got)
	}
}

// Matching is case-insensitive but the document's own casing is what gets
// rendered — a result that shouts back what the user typed is not the document.
func TestSnippetKeepsTheDocumentsCasing(t *testing.T) {
	got := Snippet("See the Runbook", []string{"runbook"}, 200)

	if want := "See the <mark>Runbook</mark>"; got != want {
		t.Errorf("Snippet() = %q, want %q", got, want)
	}
}

// Boolean mode treats +, -, * and quotes as syntax. Left in, a query like
// "C++ (draft)" is a parse error or a search for something else entirely.
func TestBooleanQueryStripsOperatorsAndUnindexableWords(t *testing.T) {
	cases := map[string]string{
		"deploy staging":  "deploy* staging*",
		"+deploy -legacy": "deploy* legacy*",
		`"exact phrase"`:  "exact* phrase*",
		"C++ (draft)":     "draft*", // "C" is below the index's minimum token
		"ci cd":           "",       // nothing indexable — the caller falls back to LIKE
	}

	for raw, want := range cases {
		if got := booleanQuery(raw, DefaultMinTokenSize); got != want {
			t.Errorf("booleanQuery(%q) = %q, want %q", raw, got, want)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
