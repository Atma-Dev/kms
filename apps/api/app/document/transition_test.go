package document

import (
	"api-kms/app"
	"api-kms/app/rbac"
	"api-kms/app/user"
	"api-kms/enums"
	"testing"

	"github.com/google/uuid"
)

func TestNextAllowsTheDocumentedLifecycle(t *testing.T) {
	cases := []struct {
		from   Status
		action Action
		want   Status
	}{
		{StatusDraft, ActionSubmitReview, StatusInReview},
		{StatusInReview, ActionApprove, StatusPublished},
		{StatusInReview, ActionReject, StatusDraft},
		{StatusDraft, ActionPublish, StatusPublished},
		{StatusPublished, ActionUnpublish, StatusDraft},
		{StatusPublished, ActionArchive, StatusArchived},
		{StatusArchived, ActionRestore, StatusDraft},
	}
	for _, c := range cases {
		got, err := Next(c.from, c.action)
		if err != nil {
			t.Errorf("%s + %s: unexpected error %v", c.from, c.action, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s + %s = %s, want %s", c.from, c.action, got, c.want)
		}
	}
}

func TestNextRefusesMovesThatSkipTheLifecycle(t *testing.T) {
	refused := []struct {
		from   Status
		action Action
	}{
		{StatusDraft, ActionApprove},         // nothing was submitted
		{StatusArchived, ActionPublish},      // must be restored first
		{StatusPublished, ActionApprove},     // already live
		{StatusDraft, ActionUnpublish},       // never was live
		{StatusInReview, ActionSubmitReview}, // already in review
		{StatusPublished, ActionRestore},     // not archived
	}
	for _, c := range refused {
		if _, err := Next(c.from, c.action); err == nil {
			t.Errorf("%s + %s was allowed, want refusal", c.from, c.action)
		}
	}
}

func TestAvailableIsStableAndMatchesNext(t *testing.T) {
	got := Available(StatusPublished)
	want := []Action{ActionUnpublish, ActionArchive}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// CanTransition is what the detail resource filters available_actions with, so
// it decides which buttons the client draws. A regression here either hides a
// move someone is entitled to or offers one the API will refuse.
func TestCanTransitionCombinesStatusAndPermission(t *testing.T) {
	published := &Document{Status: StatusPublished, AuthorID: uuid.New()}

	reviewer := withPerms("Reviewer", enums.PermissionDocumentsReview)
	if !CanTransition(published, ActionUnpublish, reviewer) {
		t.Error("a reviewer cannot unpublish a published document")
	}
	if CanTransition(published, ActionArchive, reviewer) {
		t.Error("documents.review should not carry archive rights")
	}

	archivist := withPerms("Archivist", enums.PermissionDocumentsArchive)
	if !CanTransition(published, ActionArchive, archivist) {
		t.Error("an archivist cannot archive a published document")
	}
	// Legal permission, illegal moment.
	if CanTransition(published, ActionRestore, archivist) {
		t.Error("restore was offered on a document that is not archived")
	}
}

// submit-review has no permission of its own: it belongs to whoever may edit
// the document, which is decided per row.
func TestSubmitReviewFollowsTheEditRuleNotAPermission(t *testing.T) {
	author := uuid.New()
	draft := &Document{Status: StatusDraft, AuthorID: author}

	contributor := withPerms("Contributor", enums.PermissionDocumentsUpdate)
	contributor.ID = author
	if !CanTransition(draft, ActionSubmitReview, contributor) {
		t.Error("an author cannot submit their own draft for review")
	}

	stranger := withPerms("Contributor", enums.PermissionDocumentsUpdate)
	stranger.ID = uuid.New()
	if CanTransition(draft, ActionSubmitReview, stranger) {
		t.Error("someone else's draft was submittable by a plain contributor")
	}

	editor := withPerms("Editor", enums.PermissionDocumentsUpdateAll)
	if !CanTransition(draft, ActionSubmitReview, editor) {
		t.Error("an editor cannot submit someone else's draft")
	}
}

func withPerms(roleName string, names ...enums.Permission) *user.User {
	role := rbac.Role{BaseModel: app.BaseModel{ID: uuid.New()}, Name: roleName}
	for _, name := range names {
		role.Permissions = append(role.Permissions, rbac.Permission{
			BaseModel: app.BaseModel{ID: uuid.New()}, Name: string(name),
		})
	}
	return &user.User{BaseModel: app.BaseModel{ID: uuid.New()}, Roles: []rbac.Role{role}}
}
