package category

import (
	"api-kms/app"
	"testing"

	"github.com/google/uuid"
)

func TestBuildTreeNestsAndKeepsOrder(t *testing.T) {
	root1, root2, child := uuid.New(), uuid.New(), uuid.New()
	rows := []Category{
		{BaseModel: base(root1), Name: "A"},
		{BaseModel: base(child), ParentID: &root1, Name: "A.1"},
		{BaseModel: base(root2), Name: "B"},
	}

	tree := BuildTree(rows)
	if len(tree) != 2 {
		t.Fatalf("got %d roots, want 2", len(tree))
	}
	if tree[0].Name != "A" || tree[1].Name != "B" {
		t.Errorf("roots out of order: %s, %s", tree[0].Name, tree[1].Name)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Name != "A.1" {
		t.Errorf("child not attached to A: %+v", tree[0].Children)
	}
	if len(tree[1].Children) != 0 {
		t.Errorf("B should have no children")
	}
}

func TestBuildTreeDropsOrphan(t *testing.T) {
	missing := uuid.New()
	rows := []Category{{BaseModel: base(uuid.New()), ParentID: &missing, Name: "orphan"}}

	if tree := BuildTree(rows); len(tree) != 0 {
		t.Errorf("orphan was promoted to a root: %+v", tree)
	}
}

func base(id uuid.UUID) app.BaseModel {
	return app.BaseModel{ID: id}
}

// The number beside a category has to be the number clicking it lists, so a
// parent counts everything in its subtree. A rollup that only counted direct
// children would read 0 beside a category holding a whole branch of work.
func TestRollupSumsTheWholeSubtree(t *testing.T) {
	backend, database, migrations, frontend := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	tree := BuildTree([]Category{
		{BaseModel: base(backend), Name: "Backend"},
		{BaseModel: base(database), ParentID: &backend, Name: "Database"},
		{BaseModel: base(migrations), ParentID: &database, Name: "Migrations"},
		{BaseModel: base(frontend), Name: "Frontend"},
	})

	total := Rollup(tree, map[uuid.UUID]int64{
		backend:    1, // one filed directly on the parent
		database:   2,
		migrations: 4,
		frontend:   3,
		// uuid.Nil (uncategorised) is deliberately not part of the tree total.
		uuid.Nil: 9,
	})

	if total != 10 {
		t.Errorf("tree total = %d, want 10 (uncategorised must not be included)", total)
	}

	byName := map[string]int64{}
	var walk func([]Node)
	walk = func(nodes []Node) {
		for _, n := range nodes {
			byName[n.Name] = n.DocumentCount
			walk(n.Children)
		}
	}
	walk(tree)

	for name, want := range map[string]int64{
		"Migrations": 4,     // leaf: its own
		"Database":   2 + 4, // own plus the leaf below
		"Backend":    1 + 6, // own plus the whole branch
		"Frontend":   3,
	} {
		if byName[name] != want {
			t.Errorf("%s = %d, want %d", name, byName[name], want)
		}
	}
}

// A category nobody has filed anything under reads 0, not a missing key.
func TestRollupCountsEmptyCategoriesAsZero(t *testing.T) {
	empty := uuid.New()
	tree := BuildTree([]Category{{BaseModel: base(empty), Name: "Empty"}})

	if total := Rollup(tree, map[uuid.UUID]int64{}); total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if tree[0].DocumentCount != 0 {
		t.Errorf("empty category = %d, want 0", tree[0].DocumentCount)
	}
}
