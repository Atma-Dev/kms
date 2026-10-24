// Package render lays a document out on pages and draws it, once, through
// whichever backend the protection level asks for (Modul 4 FR-2, FR-3).
//
// There is one layout engine and two canvases. That is the whole design, and
// it is the reason the strict viewer's page images and the exported PDF cannot
// drift apart: a reader who prints what they were shown gets the same page
// breaks, because the same code decided them. Two renderers written separately
// would agree on the day they were written and never again.
//
// Nothing here knows about HTTP, gorm, or what a document is — it takes blocks
// of text and gives back bytes. app/document is what turns an editor node tree
// into Blocks and decides who may ask for them.
package render

import (
	"strings"
	"unicode"
)

// Kind is the handful of block shapes a knowledge document is made of. The
// editor has more node types than this and will grow more; anything unmapped
// renders as a paragraph, which is legible rather than missing.
type Kind string

const (
	KindHeading1  Kind = "heading1"
	KindHeading2  Kind = "heading2"
	KindHeading3  Kind = "heading3"
	KindParagraph Kind = "paragraph"
	KindListItem  Kind = "list_item"
	KindQuote     Kind = "quote"
	KindCode      Kind = "code"
)

// Block is one paragraph-sized piece of the document, already flattened to the
// text a reader sees. Marks (bold, links, colour) are deliberately dropped:
// this renders a document for evidence and for reading, not a second editor,
// and every mark carried here is a mark two backends have to agree on.
type Block struct {
	Kind Kind
	Text string
}

// Options is everything a render needs beyond the blocks themselves.
type Options struct {
	// Title heads the first page.
	Title string
	// Watermark is the identity every page carries: who asked for this copy
	// and when. It is drawn tiled and diagonal across the page, and again as
	// a legible line in the footer — the tiled pattern survives a crop of the
	// footer, the footer survives a reader who cannot make the pattern out.
	//
	// An empty Watermark draws neither. Nothing in this app passes one, and
	// the zero value being "no watermark" is what keeps the tests readable.
	Watermark string
	// Scale multiplies the raster backend's resolution. 1 is 72dpi, which is
	// unreadable on screen; Pages defaults it to DefaultScale. Ignored by the
	// PDF backend, which is resolution independent.
	Scale float64
}

// A4 in points, which is the unit everything here works in. The PDF backend
// uses points natively and the raster backend multiplies by Scale, so one set
// of numbers describes both.
const (
	pageWidth  = 595.28
	pageHeight = 841.89

	marginLeft   = 56.0
	marginRight  = 56.0
	marginTop    = 64.0
	marginBottom = 64.0

	contentWidth  = pageWidth - marginLeft - marginRight
	contentBottom = pageHeight - marginBottom

	// lineSpacing multiplies the font size to give the baseline-to-baseline
	// distance.
	lineSpacing = 1.45
)

// DefaultScale renders at 144dpi: legible on a high-density screen at full
// width, and about a quarter of the bytes of the 216dpi that looks no better.
const DefaultScale = 2.0

// style is how one Kind is drawn. Kept as data rather than a switch in the
// layout loop so a new block shape is a row here, and both backends get it.
type style struct {
	size        float64
	bold        bool
	indent      float64
	spaceBefore float64
	spaceAfter  float64
	// prefix is drawn once, before the first line, and the following lines
	// align past it — a bullet that repeated on every wrapped line would read
	// as several list items.
	prefix string
	// rule draws a vertical bar down the left of the block (a quote).
	rule bool
	// mono asks the backend for its fixed-width face, so code keeps its
	// alignment.
	mono bool
}

var styles = map[Kind]style{
	KindHeading1:  {size: 20, bold: true, spaceBefore: 18, spaceAfter: 8},
	KindHeading2:  {size: 16, bold: true, spaceBefore: 14, spaceAfter: 6},
	KindHeading3:  {size: 13, bold: true, spaceBefore: 12, spaceAfter: 4},
	KindParagraph: {size: 11, spaceAfter: 8},
	KindListItem:  {size: 11, indent: 18, spaceAfter: 4, prefix: "• "},
	KindQuote:     {size: 11, indent: 16, spaceBefore: 6, spaceAfter: 8, rule: true},
	KindCode:      {size: 9.5, indent: 12, spaceBefore: 6, spaceAfter: 8, mono: true},
}

func styleFor(kind Kind) style {
	if s, ok := styles[kind]; ok {
		return s
	}
	return styles[KindParagraph]
}

// canvas is what a backend implements. Everything is in points with the origin
// at the top left of the page; a raster backend scales on its own side so the
// layout never has to know its resolution.
type canvas interface {
	// beginPage starts a fresh page, with whatever furniture (watermark,
	// footer) belongs on every one of them already drawn.
	beginPage()
	// textWidth measures a string as it would be drawn in this style.
	textWidth(s string, st style) float64
	// drawText draws one line with its baseline at y.
	drawText(s string, st style, x, y float64)
	// drawRule draws the vertical bar of a quote block.
	drawRule(x, top, bottom float64)
}

// layout walks the blocks and draws them, breaking pages as it goes.
//
// The single place page breaks are decided, which is the point: both backends
// call it with the same blocks and get the same pagination, so the PDF a
// reader exports matches the page images they were looking at.
func layout(c canvas, blocks []Block, opts Options) {
	c.beginPage()
	y := marginTop

	if opts.Title != "" {
		titleStyle := style{size: 22, bold: true, spaceAfter: 16}
		y = drawBlock(c, titleStyle, opts.Title, y)
	}

	for i, block := range blocks {
		// A document whose first line repeats its own title is the common
		// shape out of the editor — someone types the heading into the body
		// as well. Printing both puts the same words twice at the top of
		// every export, so the duplicate is dropped rather than faithfully
		// reproduced.
		if i == 0 && isTitleEcho(block, opts.Title) {
			continue
		}

		text := strings.TrimRight(block.Text, " \t")
		if strings.TrimSpace(text) == "" {
			// An empty block is vertical space in the editor, so it is
			// vertical space here — but never a page of it.
			y += styleFor(block.Kind).size
			continue
		}
		y = drawBlock(c, styleFor(block.Kind), text, y)
	}
}

// isTitleEcho reports whether a block is the document's title said again.
// Only a heading counts: a paragraph that happens to match is a sentence,
// and dropping it would silently lose a line of the document.
func isTitleEcho(block Block, title string) bool {
	if title == "" {
		return false
	}
	switch block.Kind {
	case KindHeading1, KindHeading2, KindHeading3:
		return strings.EqualFold(strings.TrimSpace(block.Text), strings.TrimSpace(title))
	}
	return false
}

// drawBlock draws one block and returns the y the next one starts at.
func drawBlock(c canvas, st style, text string, y float64) float64 {
	y += st.spaceBefore

	x := marginLeft + st.indent
	available := contentWidth - st.indent
	prefixWidth := 0.0
	if st.prefix != "" {
		prefixWidth = c.textWidth(st.prefix, st)
	}

	lines := wrap(c, text, available-prefixWidth, st)
	lineHeight := st.size * lineSpacing

	for i, line := range lines {
		if y+lineHeight > contentBottom {
			c.beginPage()
			y = marginTop
		}
		// The baseline sits a little below the top of the line box. Roughly
		// the ascent of the faces used here; exact metrics would differ per
		// backend, which is the one thing the shared layout cannot allow.
		baseline := y + st.size*0.82

		if st.rule {
			c.drawRule(marginLeft+st.indent*0.35, y, y+lineHeight)
		}
		if i == 0 && st.prefix != "" {
			c.drawText(st.prefix, st, x, baseline)
		}
		c.drawText(line, st, x+prefixWidth, baseline)
		y += lineHeight
	}

	return y + st.spaceAfter
}

// wrap is greedy word wrapping, measured through the backend's own font.
//
// A word wider than the whole line is split mid-word rather than left to run
// off the page: a pasted URL or a base64 blob is exactly the case where
// overflow would silently hide the rest of the document.
func wrap(c canvas, text string, width float64, st style) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	var current strings.Builder

	flush := func() {
		if current.Len() > 0 {
			lines = append(lines, current.String())
			current.Reset()
		}
	}

	for _, word := range strings.FieldsFunc(text, func(r rune) bool { return unicode.IsSpace(r) }) {
		candidate := word
		if current.Len() > 0 {
			candidate = current.String() + " " + word
		}
		if c.textWidth(candidate, st) <= width {
			current.Reset()
			current.WriteString(candidate)
			continue
		}

		flush()
		for c.textWidth(word, st) > width {
			cut := breakPoint(c, word, width, st)
			lines = append(lines, word[:cut])
			word = word[cut:]
		}
		current.WriteString(word)
	}
	flush()

	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// breakPoint finds how much of an over-long word fits, in whole runes. Cutting
// by bytes would split a multi-byte character into two invalid ones.
func breakPoint(c canvas, word string, width float64, st style) int {
	runes := []rune(word)
	last := len(string(runes[:1]))
	for i := 2; i <= len(runes); i++ {
		candidate := string(runes[:i])
		if c.textWidth(candidate, st) > width {
			return last
		}
		last = len(candidate)
	}
	return len(word)
}

// watermarkStep is how far apart the tiled watermark repeats, along both axes
// of the rotated grid. Close enough that a crop of any corner still carries a
// full copy, far enough that the page stays readable.
const watermarkStep = 190.0

// watermarkPositions is the grid the tiled watermark is drawn on, in page
// points. Shared by both backends so a leaked page image and a leaked PDF page
// carry the same pattern.
//
// The grid starts outside the page on both axes because the text is drawn
// rotated: a tile whose anchor sits just off the edge still puts ink on the
// page, and starting at zero would leave the top-left corner bare.
func watermarkPositions() [][2]float64 {
	var points [][2]float64
	for y := -watermarkStep; y < pageHeight+watermarkStep; y += watermarkStep {
		for x := -watermarkStep; x < pageWidth+watermarkStep; x += watermarkStep {
			points = append(points, [2]float64{x, y})
		}
	}
	return points
}

// watermarkAngle is the diagonal the PRD asks for, in degrees. Negative is
// counter-clockwise in both backends' conventions once each one's y axis is
// accounted for — see the two drawWatermark implementations.
const watermarkAngle = 45.0

const (
	watermarkSize    = 13.0
	watermarkOpacity = 0.12
	footerSize       = 7.5
)
