package document

import (
	"api-kms/app"
	"api-kms/app/rbac"
	"api-kms/app/user"
	"api-kms/enums"
	"context"
	"testing"

	"github.com/google/uuid"
)

// asReviewer is a caller holding documents.review and nothing else — in
// particular, no grant on any document.
func asReviewer(id uuid.UUID) *user.User {
	return &user.User{
		BaseModel: app.BaseModel{ID: id},
		Roles: []rbac.Role{{
			BaseModel:   app.BaseModel{ID: uuid.New()},
			Name:        "test-reviewer",
			Permissions: []rbac.Permission{{Name: string(enums.PermissionDocumentsReview)}},
		}},
	}
}

// A restricted document used to answer 404 from /documents/:id and hand its
// whole content over from /documents/:id/versions/:n — the content is the same
// content, and a restriction with a route around it is not a restriction.
//
// The three version endpoints are checked together because they leak the same
// thing three ways: the list leaks the change notes, the single version leaks
// the snapshot, and the compare leaks both versions as visible text.
func TestRestrictedDocumentDoesNotLeakThroughItsVersions(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	service := f.service()

	f.grant(t, f.locked, uuid.New(), SubjectUser, AccessView)

	doc, err := f.repo.GetById(ctx, f.locked.String())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &Version{DocumentID: doc.ID, Content: Content(`{"type":"doc"}`)}
	if err := f.repo.SaveWithSnapshot(ctx, doc, snapshot); err != nil {
		t.Fatal(err)
	}

	stranger := asUser(uuid.New())
	if _, err := service.GetById(ctx, f.locked.String(), stranger); err == nil {
		t.Fatal("the stranger can read the document itself; the rest of this test proves nothing")
	}

	if _, err := service.GetVersions(ctx, f.locked.String(), stranger); err == nil {
		t.Error("GetVersions listed the history of a document its owner is hidden from")
	}
	if _, err := service.GetVersion(ctx, f.locked.String(), snapshot.VersionNumber, stranger); err == nil {
		t.Error("GetVersion handed over a snapshot of a restricted document")
	}
	if _, err := service.Compare(ctx, f.locked.String(), "1", "current", stranger); err == nil {
		t.Error("Compare diffed a restricted document for a caller who may not read it")
	}

	// The other direction matters just as much: the check must not lock out
	// the people the grant names.
	named := uuid.New()
	f.grant(t, f.locked, named, SubjectUser, AccessView)
	if _, err := service.GetVersions(ctx, f.locked.String(), asUser(named)); err != nil {
		t.Errorf("a named subject was refused the version list: %v", err)
	}
}

// A route permission says whether a caller may make a lifecycle move at all.
// It cannot say which documents they may make it to — only the grants can, and
// publishing a restricted document is the move that undoes the restriction.
func TestRestrictedDocumentDoesNotMoveForUngrantedReviewers(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	service := f.service()

	f.grant(t, f.locked, uuid.New(), SubjectUser, AccessEdit)
	reviewer := asReviewer(uuid.New())

	if _, err := service.Transition(ctx, f.locked.String(), ActionPublish, &TransitionRequest{}, reviewer); err == nil {
		t.Error("a reviewer published a document they may not read")
	}

	// A reviewer the grant does name keeps doing their job — the check is the
	// grant, not the review permission.
	granted := uuid.New()
	f.grant(t, f.locked, granted, SubjectUser, AccessEdit)
	if _, err := service.Transition(ctx, f.locked.String(), ActionPublish, &TransitionRequest{}, asReviewer(granted)); err != nil {
		t.Errorf("a granted reviewer was refused: %v", err)
	}
}

// Same reasoning for deletion: documents.destroy says they may delete
// documents, the grants say which ones.
func TestRestrictedDocumentIsNotDeletedByUngrantedCallers(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	service := f.service()

	f.grant(t, f.locked, uuid.New(), SubjectUser, AccessEdit)

	if err := service.DeleteById(ctx, f.locked.String(), asUser(uuid.New())); err == nil {
		t.Error("a caller the document is hidden from deleted it")
	}

	// An unrestricted document is untouched by any of this.
	if err := service.DeleteById(ctx, f.open.String(), asUser(uuid.New())); err != nil {
		t.Errorf("an unrestricted document was refused: %v", err)
	}
}

// Whoever may manage access is never locked out, or a grant that named the
// wrong person would need a database client to undo.
func TestAccessManagerIsNeverLockedOut(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	service := f.service()

	f.grant(t, f.locked, uuid.New(), SubjectUser, AccessView)

	manager := &user.User{
		BaseModel: app.BaseModel{ID: uuid.New()},
		Roles: []rbac.Role{{
			BaseModel:   app.BaseModel{ID: uuid.New()},
			Name:        "test-space-admin",
			Permissions: []rbac.Permission{{Name: string(enums.PermissionDocumentsAccessManage)}},
		}},
	}

	if _, err := service.GetVersions(ctx, f.locked.String(), manager); err != nil {
		t.Errorf("an access manager was refused the version list: %v", err)
	}
	if err := service.DeleteById(ctx, f.locked.String(), manager); err != nil {
		t.Errorf("an access manager was refused the delete: %v", err)
	}
}
