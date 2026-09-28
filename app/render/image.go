package render

import (
	"bytes"
	"fmt"
	"image/jpeg"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// jpegQuality trades a little sharpness for a lot of bytes. A page of black
// text on white is mostly flat area, which JPEG handles well; at 82 the
// artefacts are invisible at reading size and the file is a third of what PNG
// would cost for the same page.
const jpegQuality = 82

// Pages renders the document as one JPEG per page, with the watermark drawn
// into the pixels (FR-3).
//
// This is what the strict viewer shows instead of the document's own text:
// there is nothing on the page to select, and the identity of whoever asked
// for it cannot be removed without removing the words too.
//
// It is not a defence against a photograph of the screen, and nothing here
// pretends otherwise.
func Pages(blocks []Block, opts Options) ([][]byte, error) {
	if opts.Scale <= 0 {
		opts.Scale = DefaultScale
	}

	c, err := newImageCanvas(opts)
	if err != nil {
		return nil, err
	}
	layout(c, blocks, opts)
	return c.pages()
}

type imageCanvas struct {
	opts  Options
	scale float64

	regular *truetype.Font
	bold    *truetype.Font
	mono    *truetype.Font
	faces   map[faceKey]font.Face

	dc    *gg.Context
	done  [][]byte
	page  int
	fatal error
}

type faceKey struct {
	size float64
	bold bool
	mono bool
}

func newImageCanvas(opts Options) (*imageCanvas, error) {
	regular, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("render: parse regular font: %w", err)
	}
	bold, err := truetype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("render: parse bold font: %w", err)
	}
	mono, err := truetype.Parse(gomono.TTF)
	if err != nil {
		return nil, fmt.Errorf("render: parse mono font: %w", err)
	}

	return &imageCanvas{
		opts:    opts,
		scale:   opts.Scale,
		regular: regular,
		bold:    bold,
		mono:    mono,
		faces:   map[faceKey]font.Face{},
	}, nil
}

// face renders a point size at the canvas's resolution by raising the DPI
// rather than the size, so every coordinate in the layout stays in points and
// only this one place knows what scale means.
func (c *imageCanvas) face(st style) font.Face {
	key := faceKey{size: st.size, bold: st.bold, mono: st.mono}
	if f, ok := c.faces[key]; ok {
		return f
	}

	source := c.regular
	switch {
	case st.mono:
		source = c.mono
	case st.bold:
		source = c.bold
	}
	f := truetype.NewFace(source, &truetype.Options{Size: st.size, DPI: 72 * c.scale})
	c.faces[key] = f
	return f
}

func (c *imageCanvas) textWidth(s string, st style) float64 {
	c.dc.SetFontFace(c.face(st))
	w, _ := c.dc.MeasureString(s)
	return w / c.scale
}

func (c *imageCanvas) drawText(s string, st style, x, y float64) {
	c.dc.SetFontFace(c.face(st))
	c.dc.SetRGB(24.0/255, 24.0/255, 27.0/255)
	c.dc.DrawString(s, x*c.scale, y*c.scale)
}

func (c *imageCanvas) drawRule(x, top, bottom float64) {
	c.dc.SetRGB(203.0/255, 213.0/255, 225.0/255)
	c.dc.DrawRectangle(x*c.scale, top*c.scale, 2.5*c.scale, (bottom-top)*c.scale)
	c.dc.Fill()
}

func (c *imageCanvas) beginPage() {
	c.finishPage()
	c.page++

	c.dc = gg.NewContext(int(pageWidth*c.scale), int(pageHeight*c.scale))
	c.dc.SetRGB(1, 1, 1)
	c.dc.Clear()
	c.drawWatermark()
	c.drawFooter()
}

func (c *imageCanvas) drawWatermark() {
	if c.opts.Watermark == "" {
		return
	}

	st := style{size: watermarkSize}
	c.dc.SetFontFace(c.face(st))
	c.dc.SetRGBA(15.0/255, 23.0/255, 42.0/255, watermarkOpacity)

	// gg's y axis points down, so a clockwise-positive rotation here draws the
	// same visual diagonal the PDF backend gets from a counter-clockwise one.
	// Both end up bottom-left to top-right; the grid is the shared one, so a
	// page and its PDF carry the same pattern in the same places.
	for _, point := range watermarkPositions() {
		x, y := point[0]*c.scale, point[1]*c.scale
		c.dc.Push()
		c.dc.RotateAbout(gg.Radians(-watermarkAngle), x, y)
		c.dc.DrawString(c.opts.Watermark, x, y)
		c.dc.Pop()
	}
}

func (c *imageCanvas) drawFooter() {
	st := style{size: footerSize}
	c.dc.SetFontFace(c.face(st))
	c.dc.SetRGB(120.0/255, 120.0/255, 128.0/255)

	y := (pageHeight - marginBottom*0.45) * c.scale
	if c.opts.Watermark != "" {
		c.dc.DrawString(c.opts.Watermark, marginLeft*c.scale, y)
	}

	number := fmt.Sprintf("%d", c.page)
	w, _ := c.dc.MeasureString(number)
	c.dc.DrawString(number, (pageWidth-marginRight)*c.scale-w, y)
}

// finishPage encodes the page in progress, if there is one. Encoding here
// rather than holding every gg.Context to the end keeps one page of pixels in
// memory at a time instead of all of them — a 40-page document at 2x is about
// 190MB of RGBA held that way and 3MB encoded.
func (c *imageCanvas) finishPage() {
	if c.dc == nil || c.fatal != nil {
		return
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, c.dc.Image(), &jpeg.Options{Quality: jpegQuality}); err != nil {
		// layout() has no error path — it is drawing, and every backend call
		// it makes returns nothing. The failure is carried here and reported
		// by pages(), which is the one place that can still refuse.
		c.fatal = fmt.Errorf("render: encode page %d: %w", c.page, err)
		return
	}
	c.done = append(c.done, out.Bytes())
}

func (c *imageCanvas) pages() ([][]byte, error) {
	c.finishPage()
	if c.fatal != nil {
		return nil, c.fatal
	}
	return c.done, nil
}
