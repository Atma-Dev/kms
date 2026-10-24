package document

import (
	"api-kms/app"
	"api-kms/app/protection"
	"api-kms/app/rbac"
	"api-kms/app/tag"
	"api-kms/app/user"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// editor holds the wide grant, so the ownership half of the edit rule is never
// what these tests are measuring.
//
// Its id is a real row: last_editor_id and document_versions.editor_id both
// carry a foreign key to users, so an invented uuid fails the write rather
// than the assertion.
func (f *fixture) anEditor() *user.User {
	return &user.User{
		BaseModel: app.BaseModel{ID: f.authorID},
		Roles: []rbac.Role{{
			BaseModel:   app.BaseModel{ID: uuid.New()},
			Name:        "Editor",
			Permissions: []rbac.Permission{{BaseModel: app.BaseModel{ID: uuid.New()}, Name: "documents.update.all"}},
		}},
	}
}

func (f *fixture) service() *Service {
	return NewService(
		f.repo,
		tag.NewService(tag.NewRepository(f.db)),
		protection.NewService(protection.NewRepository(f.db), logrus.New(), time.Hour),
	)
}

// publish marks the document as having been live, which is what turns
// versioning on at all.
func (f *fixture) publish(t *testing.T, id uuid.UUID) {
	t.Helper()
	if err := f.db.Exec("UPDATE documents SET status = 'published', published_at = ? WHERE id = ?", time.Now(), id).Error; err != nil {
		t.Fatal(err)
	}
}

// A rename is not a content change. Snapshotting it fills the version list
// with rows nothing can tell apart and buries the edits that did change
// something.
func TestRenamingAPublishedDocumentRecordsNoVersion(t *testing.T) {
	f := newFixture(t)
	f.publish(t, f.open)
	service, editor := f.service(), f.anEditor()
	ctx := context.Background()

	// Deliberately NOT doc.Content: reading it back gives MySQL's normalised
	// form, and feeding that in compares normalised against normalised — which
	// is how the first version of this test passed while renames still
	// snapshotted in the running app. This is the shape a client actually
	// sends: same meaning, different whitespace, keys in a different order.
	clientContent := Content(`{
		"content": [ {"content": [{"text": "hello", "type": "text"}], "type": "paragraph"} ],
		"type": "doc"
	}`)

	if _, err := service.UpdateById(ctx, f.open.String(),
		&UpdateDocumentRequest{Title: "first real edit", Content: clientContent}, editor); err != nil {
		t.Fatal(err)
	}
	if versions, err := f.repo.GetVersions(ctx, f.open.String()); err != nil {
		t.Fatal(err)
	} else if len(versions) != 1 {
		t.Fatalf("the content edit recorded %d versions, want 1", len(versions))
	}

	// Re-sending the same tree in a different serialisation must change
	// nothing, however many times the title moves.
	reordered := Content(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]}`)
	for _, title := range []string{"renamed once", "renamed twice", "renamed again"} {
		if _, err := service.UpdateById(ctx, f.open.String(),
			&UpdateDocumentRequest{Title: title, Content: reordered}, editor); err != nil {
			t.Fatalf("%s: %v", title, err)
		}
	}

	versions, err := f.repo.GetVersions(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Errorf("three renames took the count from 1 to %d, want it unchanged", len(versions))
	}

	// The rename itself still has to have happened.
	after, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if after.Title != "renamed again" {
		t.Errorf("title = %q, want the last rename to have stuck", after.Title)
	}
}

func TestChangingContentStillRecordsAVersion(t *testing.T) {
	f := newFixture(t)
	f.publish(t, f.open)
	service, editor := f.service(), f.anEditor()
	ctx := context.Background()

	doc, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	original := doc.Content

	if _, err := service.UpdateById(ctx, f.open.String(), &UpdateDocumentRequest{
		Title:   doc.Title,
		Content: Content(`{"type":"doc","content":[{"type":"paragraph"}]}`),
	}, editor); err != nil {
		t.Fatal(err)
	}

	versions, err := f.repo.GetVersions(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Fatalf("got %d versions, want 1", len(versions))
	}

	// The snapshot holds what was replaced, not what replaced it.
	stored, err := f.repo.GetVersion(ctx, f.open.String(), versions[0].VersionNumber)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored.Content) != string(original) {
		t.Error("the snapshot does not hold the content the edit replaced")
	}
}

// A draft nobody has seen has no reader to protect, whatever changes.
func TestEditingAnUnpublishedDraftRecordsNoVersion(t *testing.T) {
	f := newFixture(t)
	service, editor := f.service(), f.anEditor()
	ctx := context.Background()

	if _, err := service.UpdateById(ctx, f.open.String(), &UpdateDocumentRequest{
		Title:   "draft edit",
		Content: Content(`{"type":"doc","content":[{"type":"paragraph"}]}`),
	}, editor); err != nil {
		t.Fatal(err)
	}

	versions, err := f.repo.GetVersions(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 0 {
		t.Errorf("editing a draft recorded %d versions, want 0", len(versions))
	}
}

// Rolling back to what is already live changes nothing, so it records nothing
// — the same rule, applied to the other path that writes a snapshot.
func TestRollingBackToTheLiveContentRecordsNoVersion(t *testing.T) {
	f := newFixture(t)
	f.publish(t, f.open)
	service, editor := f.service(), f.anEditor()
	ctx := context.Background()

	doc, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	original := doc.Content

	// One real edit, so there is a v1 holding the original content.
	if _, err := service.UpdateById(ctx, f.open.String(), &UpdateDocumentRequest{
		Title:   doc.Title,
		Content: Content(`{"type":"doc","content":[{"type":"heading"}]}`),
	}, editor); err != nil {
		t.Fatal(err)
	}

	// Roll back to it: that is a real change, so it snapshots.
	if _, err := service.Rollback(ctx, f.open.String(), 1, &RollbackRequest{}, editor); err != nil {
		t.Fatal(err)
	}
	afterFirst, err := f.repo.GetVersions(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(afterFirst) != 2 {
		t.Fatalf("got %d versions after a real rollback, want 2", len(afterFirst))
	}

	live, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if string(live.Content) != string(original) {
		t.Fatal("the rollback did not restore v1's content")
	}

	// Rolling back to it again is a no-op and must not add a third.
	if _, err := service.Rollback(ctx, f.open.String(), 1, &RollbackRequest{}, editor); err != nil {
		t.Fatal(err)
	}
	afterSecond, err := f.repo.GetVersions(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(afterSecond) != len(afterFirst) {
		t.Errorf("a redundant rollback added a version: %d then %d", len(afterFirst), len(afterSecond))
	}
}

// Pinning must not carry the caller's copy of the content back to the
// database: a pin is a one-column write, and going through PUT would let it
// overwrite an edit someone else made in between.
func TestPinningDoesNotTouchContentOrVersions(t *testing.T) {
	f := newFixture(t)
	f.publish(t, f.open)
	service, editor := f.service(), f.anEditor()
	ctx := context.Background()

	before, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}

	pinned, err := service.SetPinned(ctx, f.open.String(), true, editor)
	if err != nil {
		t.Fatal(err)
	}
	if !pinned.IsPinned {
		t.Error("the document did not come back pinned")
	}

	after, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if !after.IsPinned {
		t.Error("is_pinned was not persisted")
	}
	if string(after.Content) != string(before.Content) || after.Title != before.Title {
		t.Error("pinning changed the content or the title")
	}

	versions, err := f.repo.GetVersions(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 0 {
		t.Errorf("pinning recorded %d versions, want 0", len(versions))
	}

	// Unpinning an unpinned document, and pinning a pinned one, are no-ops.
	if _, err := service.SetPinned(ctx, f.open.String(), true, editor); err != nil {
		t.Errorf("pinning an already-pinned document failed: %v", err)
	}
}

// A published document that was deleted comes back as a draft: restoring it
// straight to published would put content in front of readers without anyone
// looking at why it was deleted.
func TestUndeleteBringsAPublishedDocumentBackAsADraft(t *testing.T) {
	f := newFixture(t)
	f.publish(t, f.open)
	service := f.service()
	ctx := context.Background()

	if err := f.repo.SoftDeleteById(ctx, f.open.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.GetById(ctx, f.open.String()); err == nil {
		t.Fatal("the document is still visible after being deleted")
	}

	restored, err := service.Undelete(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if restored.Status != StatusDraft {
		t.Errorf("came back as %q, want draft", restored.Status)
	}

	live, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatalf("the restored document is still hidden: %v", err)
	}
	if live.Status != StatusDraft {
		t.Errorf("stored status = %q, want draft", live.Status)
	}
}

// A deleted document keeps holding its slug, so nothing can take it while it
// is in the bin and the restore cannot collide.
func TestADeletedDocumentKeepsItsSlugReserved(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	doc, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.SoftDeleteById(ctx, f.open.String()); err != nil {
		t.Fatal(err)
	}

	taken, err := f.repo.SlugExists(ctx, f.spaceID.String(), doc.Slug, "")
	if err != nil {
		t.Fatal(err)
	}
	if !taken {
		t.Error("a deleted document released its slug, so a restore could collide")
	}
}
