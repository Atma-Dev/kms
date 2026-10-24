package tag

import (
	"api-kms/internal/testdb"
	"api-kms/utils"
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seedDocument creates the minimum a document_tags row needs a foreign key
// for, and removes it again afterwards.
func seedDocument(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()

	var author struct{ ID uuid.UUID }
	if err := db.Raw("SELECT id FROM users LIMIT 1").Scan(&author).Error; err != nil || author.ID == uuid.Nil {
		t.Skip("no users in the dev database")
	}

	spaceID, documentID := uuid.New(), uuid.New()
	name := "tagtest-" + spaceID.String()[:8]

	if err := db.Exec(
		"INSERT INTO spaces (id, name, slug, is_archived, created_by, created_at, updated_at) VALUES (?,?,?,FALSE,?,NOW(),NOW())",
		spaceID, name, name, author.ID,
	).Error; err != nil {
		t.Fatalf("failed to seed space: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO documents (id, space_id, title, slug, content, status, author_id, created_at, updated_at)
		 VALUES (?,?,?,?,'{}','draft',?,NOW(),NOW())`,
		documentID, spaceID, name, name, author.ID,
	).Error; err != nil {
		t.Fatalf("failed to seed document: %v", err)
	}

	t.Cleanup(func() {
		db.Exec("DELETE FROM document_tags WHERE document_id = ?", documentID)
		db.Exec("DELETE FROM documents WHERE id = ?", documentID)
		db.Exec("DELETE FROM spaces WHERE id = ?", spaceID)
	})
	return documentID
}

// The vocabulary is open, and identity is the slug — so casing and stray
// whitespace must resolve to one tag rather than three near-duplicates.
func TestFirstOrCreateBySlugFoldsCasingAndWhitespace(t *testing.T) {
	db := testdb.Open(t)
	repo := NewRepository(db)
	ctx := context.Background()

	unique := "Deploy " + uuid.New().String()[:8]
	slug := utils.Slugify(unique)
	t.Cleanup(func() { db.Exec("DELETE FROM tags WHERE slug = ?", slug) })

	first, err := repo.FirstOrCreateBySlug(ctx, unique, slug)
	if err != nil {
		t.Fatal(err)
	}

	// A different spelling of the same word must land on the same row.
	again, err := repo.FirstOrCreateBySlug(ctx, "  "+unique+"  ", utils.Slugify("  "+unique+"  "))
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID {
		t.Errorf("a re-spelling created a second tag: %s vs %s", first.ID, again.ID)
	}

	var count int64
	db.Raw("SELECT COUNT(*) FROM tags WHERE slug = ?", slug).Scan(&count)
	if count != 1 {
		t.Errorf("got %d rows for one tag", count)
	}
}

// SyncDocumentTags replaces the whole set, so a name dropped from the list has
// to actually leave the document — an add-only sync would quietly accumulate.
func TestSyncDocumentTagsReplacesRatherThanAccumulates(t *testing.T) {
	db := testdb.Open(t)
	repo := NewRepository(db)
	service := NewService(repo)
	ctx := context.Background()

	documentID := seedDocument(t, db)
	suffix := uuid.New().String()[:8]
	a, b, c := "alpha-"+suffix, "beta-"+suffix, "gamma-"+suffix
	t.Cleanup(func() {
		db.Exec("DELETE FROM tags WHERE slug IN ?", []string{utils.Slugify(a), utils.Slugify(b), utils.Slugify(c)})
	})

	names := func(tags []Tag) []string {
		out := make([]string, len(tags))
		for i := range tags {
			out[i] = tags[i].Name
		}
		return out
	}

	got, err := service.SyncDocument(ctx, documentID, &SyncDocumentTagsRequest{Names: []string{a, b}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want two tags", names(got))
	}

	// Drop one, add one: the result is the submitted set, not the union.
	got, err = service.SyncDocument(ctx, documentID, &SyncDocumentTagsRequest{Names: []string{b, c}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want exactly the submitted set", names(got))
	}
	for _, tag := range got {
		if tag.Name == a {
			t.Errorf("%q survived a sync that did not include it", a)
		}
	}

	// An empty list clears the document.
	got, err = service.SyncDocument(ctx, documentID, &SyncDocumentTagsRequest{Names: nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("an empty sync left %v behind", names(got))
	}
}

// Two spellings of the same name in one submission are one tag — the composite
// primary key would reject the second row, so the service has to fold them.
func TestSyncDocumentTagsCollapsesDuplicatesWithinOneRequest(t *testing.T) {
	db := testdb.Open(t)
	service := NewService(NewRepository(db))
	ctx := context.Background()

	documentID := seedDocument(t, db)
	name := "dup-" + uuid.New().String()[:8]
	t.Cleanup(func() { db.Exec("DELETE FROM tags WHERE slug = ?", utils.Slugify(name)) })

	got, err := service.SyncDocument(ctx, documentID,
		&SyncDocumentTagsRequest{Names: []string{name, name, "  " + name + "  "}})
	if err != nil {
		t.Fatalf("duplicates in one request failed instead of folding: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d tags, want 1", len(got))
	}
}

// A name with nothing slug-able in it is refused rather than stored as
// "untitled": unlike a document, a tag nobody can read is not worth keeping.
func TestSyncDocumentTagsRefusesAnUnreadableName(t *testing.T) {
	db := testdb.Open(t)
	service := NewService(NewRepository(db))

	if _, err := service.SyncDocument(context.Background(), seedDocument(t, db),
		&SyncDocumentTagsRequest{Names: []string{"!!!"}}); err == nil {
		t.Error("a punctuation-only tag was accepted")
	}
}
