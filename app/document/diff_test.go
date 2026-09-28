package document

import (
	"strings"
	"testing"
)

const sample = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Panduan Cuti"}]},
  {"type":"paragraph","content":[{"type":"text","text":"Ajukan "},{"type":"text","marks":[{"type":"bold"}],"text":"14 hari"},{"type":"text","text":" sebelumnya."}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Isi formulir"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Minta approval"}]}]}
  ]}
]}`

func TestFlattenJoinsMarksAndSplitsBlocks(t *testing.T) {
	got := Flatten(Content(sample))
	want := []string{
		"Panduan Cuti",
		"Ajukan 14 hari sebelumnya.", // a bolded run stays in its paragraph's line
		"Isi formulir",               // each list item is its own line
		"Minta approval",
	}

	if len(got) != len(want) {
		t.Fatalf("got %d lines %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFlattenSurvivesGarbage(t *testing.T) {
	if got := Flatten(Content("not json")); len(got) != 1 {
		t.Errorf("got %q, want one opaque line", got)
	}
	if got := Flatten(Content("")); len(got) != 0 {
		t.Errorf("empty content produced %q", got)
	}
}

func TestDiffReportsOnlyWhatChanged(t *testing.T) {
	from := []string{"Judul", "Baris lama", "Penutup"}
	to := []string{"Judul", "Baris baru", "Penutup"}

	var ops []string
	for _, d := range Diff(from, to) {
		ops = append(ops, string(d.Op)+":"+d.Text)
	}

	got := strings.Join(ops, "|")
	want := "equal:Judul|remove:Baris lama|add:Baris baru|equal:Penutup"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDiffHandlesPureAppendAndEmptySides(t *testing.T) {
	appended := Diff([]string{"A"}, []string{"A", "B"})
	if len(appended) != 2 || appended[0].Op != OpEqual || appended[1].Op != OpAdd {
		t.Errorf("append diff = %+v", appended)
	}

	for _, d := range Diff(nil, []string{"A", "B"}) {
		if d.Op != OpAdd {
			t.Errorf("diff from empty produced %s", d.Op)
		}
	}
	for _, d := range Diff([]string{"A"}, nil) {
		if d.Op != OpRemove {
			t.Errorf("diff to empty produced %s", d.Op)
		}
	}
}

func TestDiffLineNumbersPointAtTheSideTheLineExistsOn(t *testing.T) {
	d := Diff([]string{"gone"}, []string{"new"})
	if d[0].FromLine == nil || *d[0].FromLine != 1 || d[0].ToLine != nil {
		t.Errorf("removed line numbering = %+v", d[0])
	}
	if d[1].ToLine == nil || *d[1].ToLine != 1 || d[1].FromLine != nil {
		t.Errorf("added line numbering = %+v", d[1])
	}
}

func TestExcerptCutsAtAWordBoundary(t *testing.T) {
	got := Excerpt(Content(sample), 20)
	if got == nil {
		t.Fatal("nil excerpt")
	}
	if strings.HasSuffix(*got, "…") == false {
		t.Errorf("expected an ellipsis, got %q", *got)
	}
	if strings.Contains(strings.TrimSuffix(*got, "…"), "  ") {
		t.Errorf("excerpt kept collapsed whitespace: %q", *got)
	}
	if Excerpt(Content(`{"type":"doc","content":[]}`), 20) != nil {
		t.Error("empty content should have no excerpt")
	}
}

// MySQL's JSON column sorts object keys and strips whitespace, so the stored
// form never matches the bytes a client sent. A byte comparison therefore
// reports every edit as a content change — which is exactly the bug this
// guards, and why it is worth a test of its own.
func TestSameContentIgnoresSerialisation(t *testing.T) {
	client := Content(`{
		"content": [ {"content": [{"text": "hello", "type": "text"}], "type": "paragraph"} ],
		"type": "doc"
	}`)
	stored := Content(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]}`)

	if !SameContent(client, stored) {
		t.Error("the same tree in two serialisations was reported as changed")
	}
	if !SameContent(Content(""), Content("{}")) {
		t.Error("empty content should match an empty object")
	}
}

func TestSameContentStillSeesRealChanges(t *testing.T) {
	base := Content(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]}`)

	for name, other := range map[string]Content{
		"different text":  Content(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"goodbye"}]}]}`),
		"different node":  Content(`{"type":"doc","content":[{"type":"heading","content":[{"type":"text","text":"hello"}]}]}`),
		"extra paragraph": Content(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]},{"type":"paragraph"}]}`),
	} {
		if SameContent(base, other) {
			t.Errorf("%s was reported as unchanged", name)
		}
	}

	// Unparseable counts as changed: a wasted row beats a lost snapshot.
	if SameContent(base, Content("not json")) {
		t.Error("unparseable content was reported as unchanged")
	}
}
