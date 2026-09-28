package render

import (
	"strings"
	"testing"
)

func longDocument() []Block {
	blocks := []Block{{Kind: KindHeading1, Text: "Kebijakan Keamanan Dokumen"}}
	for i := 0; i < 60; i++ {
		blocks = append(blocks,
			Block{Kind: KindParagraph, Text: strings.Repeat("Setiap salinan dokumen membawa jejak identitas pembacanya. ", 4)},
			Block{Kind: KindListItem, Text: "Watermark dinamis pada viewer dan hasil ekspor."},
		)
	}
	return blocks
}

// The invariant the whole package is built around: one layout engine means the
// PDF a reader exports breaks pages exactly where the page images they were
// looking at did. Two renderers that merely agree today would drift on the
// first change to either one.
func TestBothBackendsPaginateIdentically(t *testing.T) {
	opts := Options{Title: "Panduan KMS", Watermark: "iqbal@example.com · 2026-09-21 09:00", Scale: 1}

	pdf := newPDFCanvas(opts)
	layout(pdf, longDocument(), opts)

	img, err := newImageCanvas(opts)
	if err != nil {
		t.Fatalf("new image canvas: %v", err)
	}
	layout(img, longDocument(), opts)

	if pdf.page < 2 {
		t.Fatalf("fixture should span several pages, got %d", pdf.page)
	}
	if pdf.page != img.page {
		t.Errorf("page counts differ: pdf %d, images %d", pdf.page, img.page)
	}
}

func TestPDFIsAPDF(t *testing.T) {
	out, err := PDF(longDocument(), Options{Title: "Panduan", Watermark: "iqbal@example.com"})
	if err != nil {
		t.Fatalf("render pdf: %v", err)
	}
	if !strings.HasPrefix(string(out[:5]), "%PDF-") {
		t.Errorf("output is not a pdf, starts with %q", out[:5])
	}
}

func TestPagesAreJPEGs(t *testing.T) {
	pages, err := Pages(longDocument(), Options{Title: "Panduan", Watermark: "iqbal@example.com", Scale: 1})
	if err != nil {
		t.Fatalf("render pages: %v", err)
	}
	if len(pages) < 2 {
		t.Fatalf("expected several pages, got %d", len(pages))
	}
	for i, page := range pages {
		// JPEG's start-of-image marker. A page that encoded to nothing would
		// otherwise reach a reader as a broken image with no error anywhere.
		if len(page) < 2 || page[0] != 0xFF || page[1] != 0xD8 {
			t.Errorf("page %d is not a jpeg (%d bytes)", i, len(page))
		}
	}
}

// A pasted URL or a base64 blob is one "word" wider than the page. Letting it
// run off the edge would silently hide the rest of the line.
func TestWrapSplitsAnOverlongWord(t *testing.T) {
	c := newPDFCanvas(Options{})
	c.beginPage()

	st := styleFor(KindParagraph)
	word := strings.Repeat("a", 500)
	lines := wrap(c, word, contentWidth, st)

	if len(lines) < 2 {
		t.Fatalf("expected the word to be split, got %d line(s)", len(lines))
	}
	if strings.Join(lines, "") != word {
		t.Error("splitting lost or duplicated characters")
	}
	for i, line := range lines {
		if w := c.textWidth(line, st); w > contentWidth {
			t.Errorf("line %d is %.1fpt wide, over the %.1fpt content width", i, w, contentWidth)
		}
	}
}

// Multi-byte characters must not be cut in half — a break by bytes would leave
// two invalid runes where one character was.
func TestWrapSplitsOnRuneBoundaries(t *testing.T) {
	c := newPDFCanvas(Options{})
	c.beginPage()

	word := strings.Repeat("あ", 300)
	joined := strings.Join(wrap(c, word, contentWidth, styleFor(KindParagraph)), "")
	if joined != word {
		t.Errorf("rune boundaries not respected: got %d runes, want %d", len([]rune(joined)), len([]rune(word)))
	}
}

func TestResolveFallsBackToTheSpaceThenToNormal(t *testing.T) {
	// Guards the layout's own defaulting, which is the only rule in here that
	// is not visible in the output.
	if got := styleFor("something-the-editor-added-later"); got.size != styles[KindParagraph].size {
		t.Errorf("unknown block kind should render as a paragraph, got size %v", got.size)
	}
}

// The editor's common shape is a document whose first line repeats its own
// title. Printing both would put the same words twice at the top of every
// export and every page image.
func TestLayoutDropsATitleSaidTwice(t *testing.T) {
	opts := Options{Title: "Panduan Deploy"}
	blocks := []Block{
		{Kind: KindHeading1, Text: "Panduan Deploy"},
		{Kind: KindParagraph, Text: "Deploy ke staging lewat pipeline."},
	}

	withEcho := newPDFCanvas(opts)
	layout(withEcho, blocks, opts)

	withoutEcho := newPDFCanvas(opts)
	layout(withoutEcho, blocks[1:], opts)

	if withEcho.page != withoutEcho.page {
		t.Errorf("dropping the echo should change nothing but the duplicate line")
	}

	// A paragraph that happens to match is a sentence, not a repeated title.
	if isTitleEcho(Block{Kind: KindParagraph, Text: "Panduan Deploy"}, "Panduan Deploy") {
		t.Error("only a heading may be treated as the title said again")
	}
	if !isTitleEcho(Block{Kind: KindHeading1, Text: " panduan deploy "}, "Panduan Deploy") {
		t.Error("the comparison should ignore case and surrounding space")
	}
}
