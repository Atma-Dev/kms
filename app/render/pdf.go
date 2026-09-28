package render

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// The three faces, registered from bytes compiled into the binary rather than
// read from disk. A PDF that depends on a font file is a PDF that renders
// differently — or not at all — depending on what the container image happens
// to ship.
//
// The same three faces back the raster canvas, so a page and its PDF break
// lines identically.
const (
	fontBody = "body"
	fontMono = "mono"
)

// PDF renders the document as text on pages (FR-2), watermarked with
// opts.Watermark.
//
// The text is real text: selectable, searchable, and reachable by a screen
// reader. That is the right trade at the normal protection level, where the
// point of the watermark is to trace a copy rather than to prevent one.
// ImagePDF is the strict counterpart.
func PDF(blocks []Block, opts Options) ([]byte, error) {
	c := newPDFCanvas(opts)
	layout(c, blocks, opts)
	return c.bytes()
}

// ImagePDF renders the document as one full-page image per page (FR-3).
//
// Nothing in the result is text, so nothing can be selected or copied out of
// it, and the watermark is in the same pixels as the words — a reader who
// strips it strips the document with it. The cost is a file several times
// larger that no screen reader can read, which is why it is the strict level
// and not the default.
func ImagePDF(blocks []Block, opts Options) ([]byte, error) {
	pages, err := Pages(blocks, opts)
	if err != nil {
		return nil, err
	}

	pdf := newDocument()
	for i, page := range pages {
		pdf.AddPage()
		name := fmt.Sprintf("page-%d", i)
		options := fpdf.ImageOptions{ImageType: "JPG"}
		pdf.RegisterImageOptionsReader(name, options, bytes.NewReader(page))
		// Full bleed: the image was rendered at exactly this page's
		// proportions, so it neither stretches nor leaves a margin.
		pdf.ImageOptions(name, 0, 0, pageWidth, pageHeight, false, options, 0, "")
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("render: write image pdf: %w", err)
	}
	return out.Bytes(), nil
}

func newDocument() *fpdf.Fpdf {
	pdf := fpdf.New("P", "pt", "A4", "")
	// This package paginates in layout(), which is what keeps the PDF and the
	// page images agreeing. fpdf breaking pages on its own would put the two
	// backends one line out of step on any page that fills.
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddUTF8FontFromBytes(fontBody, "", goregular.TTF)
	pdf.AddUTF8FontFromBytes(fontBody, "B", gobold.TTF)
	pdf.AddUTF8FontFromBytes(fontMono, "", gomono.TTF)
	return pdf
}

type pdfCanvas struct {
	pdf  *fpdf.Fpdf
	opts Options
	page int
}

func newPDFCanvas(opts Options) *pdfCanvas {
	return &pdfCanvas{pdf: newDocument(), opts: opts}
}

func (c *pdfCanvas) setFont(st style) {
	family := fontBody
	weight := ""
	if st.mono {
		family = fontMono
	} else if st.bold {
		weight = "B"
	}
	c.pdf.SetFont(family, weight, st.size)
}

func (c *pdfCanvas) textWidth(s string, st style) float64 {
	c.setFont(st)
	return c.pdf.GetStringWidth(s)
}

func (c *pdfCanvas) drawText(s string, st style, x, y float64) {
	c.setFont(st)
	c.pdf.SetTextColor(24, 24, 27)
	c.pdf.Text(x, y, s)
}

func (c *pdfCanvas) drawRule(x, top, bottom float64) {
	c.pdf.SetFillColor(203, 213, 225)
	c.pdf.Rect(x, top, 2.5, bottom-top, "F")
}

func (c *pdfCanvas) beginPage() {
	c.page++
	c.pdf.AddPage()
	c.drawWatermark()
	c.drawFooter()
}

func (c *pdfCanvas) drawWatermark() {
	if c.opts.Watermark == "" {
		return
	}

	// Drawn before the content, so the words sit on top of the pattern rather
	// than under it. On a page this is only legibility; on a raster page it is
	// the same pixels either way.
	c.pdf.SetAlpha(watermarkOpacity, "")
	c.pdf.SetFont(fontBody, "", watermarkSize)
	c.pdf.SetTextColor(15, 23, 42)
	for _, point := range watermarkPositions() {
		c.pdf.TransformBegin()
		c.pdf.TransformRotate(watermarkAngle, point[0], point[1])
		c.pdf.Text(point[0], point[1], c.opts.Watermark)
		c.pdf.TransformEnd()
	}
	c.pdf.SetAlpha(1, "")
}

// drawFooter writes the watermark once more, legibly, beside the page number.
//
// The tiled pattern survives someone cropping the footer away; this survives
// someone photographing the page at an angle that makes the faint diagonal
// unreadable. Neither is redundant with the other.
func (c *pdfCanvas) drawFooter() {
	c.pdf.SetFont(fontBody, "", footerSize)
	c.pdf.SetTextColor(120, 120, 128)

	y := pageHeight - marginBottom*0.45
	if c.opts.Watermark != "" {
		c.pdf.Text(marginLeft, y, c.opts.Watermark)
	}
	number := fmt.Sprintf("%d", c.page)
	c.pdf.Text(pageWidth-marginRight-c.pdf.GetStringWidth(number), y, number)
}

func (c *pdfCanvas) bytes() ([]byte, error) {
	var out bytes.Buffer
	if err := c.pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("render: write pdf: %w", err)
	}
	return out.Bytes(), nil
}
