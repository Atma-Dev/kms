package attachment

import (
	"api-kms/config"
	"api-kms/internal/testdb"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seedUpload writes a real file and the row that names it, so the job is
// exercised against both halves of what it has to clean up.
func seedUpload(t *testing.T, db *gorm.DB, root string, kind Kind, documentID *uuid.UUID, age time.Duration) (uuid.UUID, string) {
	t.Helper()

	var uploader struct{ ID uuid.UUID }
	if err := db.Raw("SELECT id FROM users LIMIT 1").Scan(&uploader).Error; err != nil || uploader.ID == uuid.Nil {
		t.Skip("no users in the dev database")
	}

	id, space := uuid.New(), uuid.New()
	relative := relativePath(space, kind, id.String()+".png")
	absolute := filepath.Join(root, filepath.FromSlash(relative))

	if err := os.MkdirAll(filepath.Dir(absolute), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absolute, []byte("not really a png"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(
		`INSERT INTO attachments (id, document_id, kind, file_path, file_name, file_type, file_size, uploaded_by, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,NOW())`,
		id, documentID, kind, relative, "x.png", "image/png", 16, uploader.ID, time.Now().Add(-age),
	).Error; err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { db.Exec("DELETE FROM attachments WHERE id = ?", id) })
	return id, absolute
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// The job's whole reason to exist: an inline image dropped into an editor and
// then abandoned belongs to no document, so nothing else ever removes it.
func TestPruneRemovesAbandonedInlineUploadsAndTheirFiles(t *testing.T) {
	db := testdb.Open(t)
	root := t.TempDir()
	config.AppConfig = &config.Config{StoragePath: root, UploadOrphanRetentionHours: 24}

	old, oldFile := seedUpload(t, db, root, KindInline, nil, 48*time.Hour)

	message, err := NewPruneOrphanUploadsJob(NewRepository(db)).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(message)

	var count int64
	db.Raw("SELECT COUNT(*) FROM attachments WHERE id = ?", old).Scan(&count)
	if count != 0 {
		t.Error("the abandoned upload's row survived the sweep")
	}
	if exists(oldFile) {
		t.Error("the row went but the file was left on disk")
	}
}

// Three things must survive: an upload still inside the writing session, one
// that reached a document, and a plain attachment.
func TestPruneLeavesEverythingItShould(t *testing.T) {
	db := testdb.Open(t)
	root := t.TempDir()
	config.AppConfig = &config.Config{StoragePath: root, UploadOrphanRetentionHours: 24}

	// A document to anchor the non-orphans to.
	var author struct{ ID uuid.UUID }
	if err := db.Raw("SELECT id FROM users LIMIT 1").Scan(&author).Error; err != nil || author.ID == uuid.Nil {
		t.Skip("no users in the dev database")
	}
	spaceID, documentID := uuid.New(), uuid.New()
	name := "prunetest-" + spaceID.String()[:8]
	if err := db.Exec("INSERT INTO spaces (id, name, slug, is_archived, created_by, created_at, updated_at) VALUES (?,?,?,FALSE,?,NOW(),NOW())",
		spaceID, name, name, author.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO documents (id, space_id, title, slug, content, status, author_id, created_at, updated_at)
	                   VALUES (?,?,?,?,'{}','draft',?,NOW(),NOW())`,
		documentID, spaceID, name, name, author.ID).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM attachments WHERE document_id = ?", documentID)
		db.Exec("DELETE FROM documents WHERE id = ?", documentID)
		db.Exec("DELETE FROM spaces WHERE id = ?", spaceID)
	})

	// Still being written: younger than the retention window.
	fresh, freshFile := seedUpload(t, db, root, KindInline, nil, time.Hour)
	// Made it into a document, however old.
	used, usedFile := seedUpload(t, db, root, KindInline, &documentID, 72*time.Hour)
	// A downloadable attachment is never this job's business.
	file, fileFile := seedUpload(t, db, root, KindAttachment, &documentID, 72*time.Hour)

	if _, err := NewPruneOrphanUploadsJob(NewRepository(db)).Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	for label, pair := range map[string]struct {
		id   uuid.UUID
		path string
	}{
		"an upload still inside the writing session": {fresh, freshFile},
		"an inline image that reached a document":    {used, usedFile},
		"a downloadable attachment":                  {file, fileFile},
	} {
		var count int64
		db.Raw("SELECT COUNT(*) FROM attachments WHERE id = ?", pair.id).Scan(&count)
		if count != 1 {
			t.Errorf("%s was pruned", label)
		}
		if !exists(pair.path) {
			t.Errorf("%s lost its file", label)
		}
	}
}

// A file that cannot be removed must not take the sweep down with it — the
// next hour retries, and one unreadable path should not strand everything
// behind it.
func TestPruneKeepsGoingWhenAFileCannotBeRemoved(t *testing.T) {
	db := testdb.Open(t)
	root := t.TempDir()
	config.AppConfig = &config.Config{StoragePath: root, UploadOrphanRetentionHours: 24}

	broken, brokenFile := seedUpload(t, db, root, KindInline, nil, 48*time.Hour)
	// A directory where a file should be: os.Remove refuses it.
	if err := os.Remove(brokenFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(brokenFile, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenFile, "blocker"), []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}

	good, goodFile := seedUpload(t, db, root, KindInline, nil, 48*time.Hour)

	message, err := NewPruneOrphanUploadsJob(NewRepository(db)).Run(context.Background())
	if err != nil {
		t.Fatalf("one bad file failed the whole run: %v", err)
	}
	t.Log(message)

	var goodCount, brokenCount int64
	db.Raw("SELECT COUNT(*) FROM attachments WHERE id = ?", good).Scan(&goodCount)
	db.Raw("SELECT COUNT(*) FROM attachments WHERE id = ?", broken).Scan(&brokenCount)

	if goodCount != 0 || exists(goodFile) {
		t.Error("the removable orphan was not cleaned up")
	}
	if brokenCount != 1 {
		t.Error("the row was dropped even though its file could not be removed, stranding the file forever")
	}
}
