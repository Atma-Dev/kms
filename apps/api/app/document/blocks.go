package document

import (
	"api-kms/app/render"
	"encoding/json"
)

// Blocks turns the editor's node tree into the shapes app/render draws
// (Modul 4 FR-2, FR-3).
//
// Flatten, beside it in diff.go, answers a different question: it returns the
// text a reviewer compares, one line per block and nothing else. Rendering
// needs to know a heading from a bullet, so this walk keeps the node type that
// one deliberately discards. They stay separate rather than one function with
// a flag — a diff that started reporting "this paragraph became a heading" as
// a text change would be a regression in the reviewer's tool.
//
// Marks (bold, italic, links) are dropped here as they are there. A rendered
// page is for reading and for evidence; carrying marks would mean both
// backends agreeing on how to draw each one, and the watermark is what the
// page is actually for.
func Blocks(content Content) []render.Block {
	var root any
	if err := json.Unmarshal(normalizeContent(content), &root); err != nil {
		// Unparseable content renders as its own raw text rather than as an
		// error. A reader asking for a PDF of a document with one malformed
		// field is better served by something they can read than by a 500.
		return []render.Block{{Kind: render.KindParagraph, Text: string(content)}}
	}

	var blocks []render.Block
	collectBlocks(root, "", &blocks)
	return blocks
}

// collectBlocks walks the tree the same way collect does, carrying the kind
// imposed by an enclosing node.
//
// The enclosing kind is what makes a quote a quote: in ProseMirror a
// blockquote holds paragraphs, and it is the paragraph that carries the text.
// Without the context every quote would render as an ordinary paragraph and
// every list item would lose its bullet.
func collectBlocks(node any, inherited render.Kind, blocks *[]render.Block) {
	switch n := node.(type) {
	case []any:
		for _, child := range n {
			collectBlocks(child, inherited, blocks)
		}
	case map[string]any:
		nodeType, _ := n["type"].(string)
		context := inherited
		switch nodeType {
		case "blockquote":
			context = render.KindQuote
		case "listItem":
			context = render.KindListItem
		case "codeBlock":
			context = render.KindCode
		}

		if text := inlineText(n); text != "" {
			*blocks = append(*blocks, render.Block{Kind: kindOf(nodeType, n, context), Text: text})
			return
		}

		if content, ok := n["content"]; ok {
			collectBlocks(content, context, blocks)
			return
		}

		// A paragraph with no content at all is the blank line someone typed
		// between two sections. It survives into the render as the vertical
		// space it is; every other empty node is structure and is dropped.
		if nodeType == "paragraph" {
			*blocks = append(*blocks, render.Block{Kind: render.KindParagraph})
		}
	}
}

// kindOf decides how one text-carrying node is drawn.
//
// A heading wins over its context: a heading inside a list item is still a
// heading, and rendering it as a bullet would lose the structure the author
// chose. Everything else defers to the context, which is what carries "this
// paragraph is inside a quote" down to the node that holds the words.
func kindOf(nodeType string, node map[string]any, context render.Kind) render.Kind {
	if nodeType == "heading" {
		switch headingLevel(node) {
		case 1:
			return render.KindHeading1
		case 2:
			return render.KindHeading2
		default:
			// Level 3 and anything deeper. A document nesting six levels is
			// not served by six sizes, and the last two would be
			// indistinguishable from body text at any readable size.
			return render.KindHeading3
		}
	}
	if context != "" {
		return context
	}
	if nodeType == "codeBlock" {
		return render.KindCode
	}
	return render.KindParagraph
}

// headingLevel reads attrs.level, which arrives as a JSON number. A heading
// with no level is level 1 — that is what the editor means by omitting it.
func headingLevel(node map[string]any) int {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return 1
	}
	level, ok := attrs["level"].(float64)
	if !ok {
		return 1
	}
	return int(level)
}
