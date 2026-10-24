package document

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Flatten walks the editor's node tree and returns one line per block: the
// text a reader would see, in reading order.
//
// Diffing the raw JSON would report a change every time an attribute the
// reader cannot see moves, and diffing rendered HTML would report every markup
// change as a text change. The visible text is what a reviewer is comparing.
//
// Unknown node types are walked like any other — the rule is "a node's text is
// its own text plus its children's", which holds for every ProseMirror node
// without this needing to know their names.
func Flatten(content Content) []string {
	var root any
	if err := json.Unmarshal(normalizeContent(content), &root); err != nil {
		// Unparseable content is one opaque line rather than an error: a diff
		// is a read, and failing the whole comparison because one old snapshot
		// is malformed helps nobody.
		return []string{string(content)}
	}

	var lines []string
	collect(root, &lines)
	return lines
}

// collect appends one line per block-level node. A node is block-level here if
// it has children that produced text — leaf text is joined into its parent's
// line, which is what keeps a bolded word from becoming its own line.
func collect(node any, lines *[]string) {
	switch n := node.(type) {
	case []any:
		for _, child := range n {
			collect(child, lines)
		}
	case map[string]any:
		if text := inlineText(n); text != "" {
			*lines = append(*lines, text)
			return
		}
		// No inline text of its own: recurse so nested blocks (list items,
		// table cells, blockquotes) each get their own line.
		if content, ok := n["content"]; ok {
			collect(content, lines)
		}
	}
}

// inlineText joins a node's immediate text descendants, stopping at any child
// that is itself a block. Returns "" when the node has none, which is what
// tells collect to recurse instead.
func inlineText(node map[string]any) string {
	if text, ok := node["text"].(string); ok {
		return text
	}

	content, ok := node["content"].([]any)
	if !ok {
		return ""
	}

	var b strings.Builder
	for _, child := range content {
		childMap, ok := child.(map[string]any)
		if !ok {
			continue
		}
		// A child with its own content is a block — this node is a container,
		// not a paragraph, so it gets no line of its own.
		if _, nested := childMap["content"]; nested {
			return ""
		}
		b.WriteString(inlineText(childMap))
	}
	return strings.TrimSpace(b.String())
}

func normalizeContent(content Content) []byte {
	if len(content) == 0 {
		return []byte("{}")
	}
	return content
}

// Op is what happened to one line between two versions.
type Op string

const (
	OpEqual  Op = "equal"
	OpAdd    Op = "add"
	OpRemove Op = "remove"
)

// DiffLine is one line of the comparison, carrying its position in whichever
// side it exists on so the client can render two gutters without recounting.
type DiffLine struct {
	Op   Op     `json:"op"`
	Text string `json:"text"`
	// FromLine / ToLine are 1-based, and nil on the side the line is absent
	// from.
	FromLine *int `json:"from_line"`
	ToLine   *int `json:"to_line"`
}

// Diff compares two flattened versions line by line.
//
// Plain LCS, no dependency: the input is a handful of paragraphs, and the
// alternatives all pull in a diff library to produce the same answer for this
// size of input.
//
// ponytail: O(n·m) time and memory in the line count. A 5,000-line document
// costs ~25M cells; if documents ever get that long, switch to Myers.
func Diff(from, to []string) []DiffLine {
	lcs := lcsTable(from, to)

	result := make([]DiffLine, 0, len(from)+len(to))
	i, j := 0, 0
	for i < len(from) && j < len(to) {
		switch {
		case from[i] == to[j]:
			result = append(result, DiffLine{Op: OpEqual, Text: from[i], FromLine: line(i), ToLine: line(j)})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			result = append(result, DiffLine{Op: OpRemove, Text: from[i], FromLine: line(i)})
			i++
		default:
			result = append(result, DiffLine{Op: OpAdd, Text: to[j], ToLine: line(j)})
			j++
		}
	}
	for ; i < len(from); i++ {
		result = append(result, DiffLine{Op: OpRemove, Text: from[i], FromLine: line(i)})
	}
	for ; j < len(to); j++ {
		result = append(result, DiffLine{Op: OpAdd, Text: to[j], ToLine: line(j)})
	}

	return result
}

// lcsTable[i][j] is the longest common subsequence length of from[i:] and
// to[j:]. Built from the end so the walk above can read it forwards.
func lcsTable(from, to []string) [][]int {
	table := make([][]int, len(from)+1)
	for i := range table {
		table[i] = make([]int, len(to)+1)
	}

	for i := len(from) - 1; i >= 0; i-- {
		for j := len(to) - 1; j >= 0; j-- {
			if from[i] == to[j] {
				table[i][j] = table[i+1][j+1] + 1
			} else if table[i+1][j] >= table[i][j+1] {
				table[i][j] = table[i+1][j]
			} else {
				table[i][j] = table[i][j+1]
			}
		}
	}
	return table
}

func line(index int) *int {
	n := index + 1
	return &n
}

// PlainText is the whole document as one line of readable text: what Flatten
// produces, with the blocks joined and the whitespace normalised.
//
// Two readers, both of which want the same string. Excerpt cuts its opening
// for the preview; documents.search_body stores all of it for the full-text
// index (Modul 3 FR-1), which is why this is not just Excerpt with a bigger
// max — an index wants the tail of a long document as much as its head.
func PlainText(content Content) string {
	return strings.Join(strings.Fields(strings.Join(Flatten(content), " ")), " ")
}

// Excerpt is the plain-text preview stored on the document: the opening of its
// content, cut at a word boundary.
func Excerpt(content Content, max int) *string {
	text := PlainText(content)
	if text == "" {
		return nil
	}

	if len(text) > max {
		cut := text[:max]
		// Cutting mid-word reads like a typo; cutting at the last space reads
		// like a preview.
		if space := strings.LastIndex(cut, " "); space > 0 {
			cut = cut[:space]
		}
		text = cut + "…"
	}

	return &text
}

// SameContent reports whether two node trees mean the same thing.
//
// Not a byte comparison. A JSONB column is a normalising store: it strips
// whitespace and sorts object keys, so what comes back out never matches the
// bytes a client sent even when nothing changed. A byte check therefore calls
// every edit a content change, which is the bug it was meant to prevent.
//
// Unparseable content counts as changed. That direction is the safe one: an
// unnecessary snapshot wastes a row, a missed one loses the content an edit
// replaced.
func SameContent(a, b Content) bool {
	var left, right any
	if err := json.Unmarshal(normalizeContent(a), &left); err != nil {
		return false
	}
	if err := json.Unmarshal(normalizeContent(b), &right); err != nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}
