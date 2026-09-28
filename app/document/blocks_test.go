package document

import (
	"api-kms/app/render"
	"testing"
)

func TestBlocksKeepsTheShapeFlattenDiscards(t *testing.T) {
	content := Content(`{"type":"doc","content":[
		{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Tujuan"}]},
		{"type":"paragraph","content":[{"type":"text","text":"Melacak sumber salinan."}]},
		{"type":"bulletList","content":[
			{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Watermark dinamis"}]}]},
			{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Access log"}]}]}
		]},
		{"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"Bukan DRM penuh."}]}]},
		{"type":"codeBlock","content":[{"type":"text","text":"GET /documents/1/export-pdf"}]}
	]}`)

	want := []render.Block{
		{Kind: render.KindHeading2, Text: "Tujuan"},
		{Kind: render.KindParagraph, Text: "Melacak sumber salinan."},
		{Kind: render.KindListItem, Text: "Watermark dinamis"},
		{Kind: render.KindListItem, Text: "Access log"},
		{Kind: render.KindQuote, Text: "Bukan DRM penuh."},
		{Kind: render.KindCode, Text: "GET /documents/1/export-pdf"},
	}

	got := Blocks(content)
	if len(got) != len(want) {
		t.Fatalf("got %d blocks, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("block %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// A heading inside a list item is still a heading: the context carries "this
// is a bullet" down to paragraphs, but must not overwrite a shape the author
// picked explicitly.
func TestBlocksLetsAHeadingWinOverItsContext(t *testing.T) {
	content := Content(`{"type":"doc","content":[
		{"type":"listItem","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Judul"}]}]}
	]}`)

	got := Blocks(content)
	if len(got) != 1 || got[0].Kind != render.KindHeading1 {
		t.Errorf("got %+v, want one heading1", got)
	}
}

func TestBlocksSurvivesMalformedContent(t *testing.T) {
	got := Blocks(Content(`{"type":"doc",`))
	if len(got) != 1 || got[0].Text == "" {
		t.Errorf("malformed content should render as its own text, got %+v", got)
	}
}
