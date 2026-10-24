package document

import (
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	"api-kms/app"
	"api-kms/app/rbac"
	"api-kms/app/user"
	"api-kms/internal/testdb"
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// fixture builds a throwaway space with two documents and removes everything
// it created, so a run leaves the dev database exactly as it found it.
type fixture struct {
	db       *gorm.DB
	repo     *Repository
	spaceID  uuid.UUID
	authorID uuid.UUID
	open     uuid.UUID // no access grants
	locked   uuid.UUID // restricted once a grant is added
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testdb.Open(t)

	// Any existing user will do as the author; the FK is all that matters.
	var author user.User
	if err := db.First(&author).Error; err != nil {
		t.Skipf("no users in the dev database: %v", err)
	}

	f := &fixture{db: db, repo: NewRepository(db), spaceID: uuid.New(), authorID: author.ID, open: uuid.New(), locked: uuid.New()}

	if err := db.Exec(
		"INSERT INTO spaces (id, name, slug, is_archived, created_by, created_at, updated_at) VALUES (?,?,?,0,?,NOW(),NOW())",
		f.spaceID, "test-"+f.spaceID.String()[:8], "test-"+f.spaceID.String()[:8], author.ID,
	).Error; err != nil {
		t.Fatalf("failed to seed space: %v", err)
	}

	for _, id := range []uuid.UUID{f.open, f.locked} {
		if err := db.Exec(
			`INSERT INTO documents (id, space_id, title, slug, content, status, author_id, created_at, updated_at)
			 VALUES (?,?,?,?,'{}','draft',?,NOW(),NOW())`,
			id, f.spaceID, "doc-"+id.String()[:8], "doc-"+id.String()[:8], author.ID,
		).Error; err != nil {
			t.Fatalf("failed to seed document: %v", err)
		}
	}

	t.Cleanup(func() {
		db.Exec("DELETE FROM document_access WHERE document_id IN ?", []uuid.UUID{f.open, f.locked})
		db.Exec("DELETE FROM document_versions WHERE document_id IN ?", []uuid.UUID{f.open, f.locked})
		db.Exec("DELETE FROM documents WHERE space_id = ?", f.spaceID)
		db.Exec("DELETE FROM spaces WHERE id = ?", f.spaceID)
	})

	return f
}

func (f *fixture) grant(t *testing.T, documentID, subjectID uuid.UUID, subjectType SubjectType, level AccessLevel) {
	t.Helper()
	if err := f.db.Exec(
		"INSERT INTO document_access (id, document_id, subject_type, subject_id, permission, created_at, updated_at) VALUES (?,?,?,?,?,NOW(),NOW())",
		uuid.New(), documentID, subjectType, subjectID, level,
	).Error; err != nil {
		t.Fatalf("failed to seed grant: %v", err)
	}
}

func asUser(id uuid.UUID, roleIDs ...uuid.UUID) *user.User {
	u := &user.User{BaseModel: app.BaseModel{ID: id}}
	for _, roleID := range roleIDs {
		u.Roles = append(u.Roles, rbac.Role{BaseModel: app.BaseModel{ID: roleID}})
	}
	return u
}

// RestrictedIDsExcluding is the query behind FR-9's list filtering. It is the
// one place a mistake is silent in both directions: too eager and a document
// vanishes for everyone, too lax and a restricted one is listed to strangers.
func TestRestrictedIDsExcludingHidesOnlyWhatItShould(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	stranger := asUser(uuid.New())

	hidden, err := f.repo.RestrictedIDsExcluding(ctx, f.spaceID.String(), stranger)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden) != 0 {
		t.Fatalf("nothing is restricted yet, but %d ids were hidden", len(hidden))
	}

	named := uuid.New()
	f.grant(t, f.locked, named, SubjectUser, AccessView)

	hidden, err = f.repo.RestrictedIDsExcluding(ctx, f.spaceID.String(), stranger)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden) != 1 || hidden[0] != f.locked {
		t.Errorf("stranger should have exactly the locked document hidden, got %v", hidden)
	}

	hidden, err = f.repo.RestrictedIDsExcluding(ctx, f.spaceID.String(), asUser(named))
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden) != 0 {
		t.Errorf("the named user should see everything, got %v hidden", hidden)
	}
}

// A grant on a role has to reach whoever holds it at read time — staffing
// changes should not need the grants rewritten.
func TestRestrictedIDsExcludingFollowsRoleGrants(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	roleID := uuid.New()
	f.grant(t, f.locked, roleID, SubjectRole, AccessView)

	holder := asUser(uuid.New(), roleID)
	hidden, err := f.repo.RestrictedIDsExcluding(ctx, f.spaceID.String(), holder)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden) != 0 {
		t.Errorf("a role holder should not be hidden from, got %v", hidden)
	}

	outsider := asUser(uuid.New(), uuid.New())
	hidden, err = f.repo.RestrictedIDsExcluding(ctx, f.spaceID.String(), outsider)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden) != 1 {
		t.Errorf("someone without the role should be hidden from it, got %v", hidden)
	}
}

// The snapshot and the write share one transaction, and version numbers are
// per document. Getting either wrong loses the content an edit replaced, which
// is the one thing versioning exists to prevent.
func TestSaveWithSnapshotNumbersPerDocumentAndMovesThePointer(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	doc, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}

	for i, want := range []uint{1, 2, 3} {
		snapshot := &Version{DocumentID: doc.ID, Content: Content(`{"v":` + string(rune('0'+i)) + `}`)}
		if err := f.repo.SaveWithSnapshot(ctx, doc, snapshot); err != nil {
			t.Fatalf("snapshot %d failed: %v", want, err)
		}
		if snapshot.VersionNumber != want {
			t.Errorf("version number = %d, want %d", snapshot.VersionNumber, want)
		}
		if doc.CurrentVersionID == nil || *doc.CurrentVersionID != snapshot.ID {
			t.Errorf("current_version_id does not point at the snapshot just taken")
		}
	}

	// A second document starts its own numbering at 1 — "v1" must mean the
	// same thing in the UI as it does here.
	other, err := f.repo.GetById(ctx, f.locked.String())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &Version{DocumentID: other.ID, Content: Content(`{}`)}
	if err := f.repo.SaveWithSnapshot(ctx, other, snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.VersionNumber != 1 {
		t.Errorf("a second document started at v%d, want v1", snapshot.VersionNumber)
	}

	versions, err := f.repo.GetVersions(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 3 {
		t.Errorf("got %d versions, want 3", len(versions))
	}
	// Newest first, and without content — the list is a sidebar of numbers.
	if len(versions) == 3 && versions[0].VersionNumber != 3 {
		t.Errorf("versions are not newest-first: %d", versions[0].VersionNumber)
	}
}

// A nil snapshot is the draft case: nothing is recorded, and the pointer stays
// where it was.
func TestSaveWithoutSnapshotRecordsNoVersion(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	doc, err := f.repo.GetById(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	doc.Title = "renamed"
	if err := f.repo.SaveWithSnapshot(ctx, doc, nil); err != nil {
		t.Fatal(err)
	}

	versions, err := f.repo.GetVersions(ctx, f.open.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 0 {
		t.Errorf("an unpublished edit recorded %d versions, want 0", len(versions))
	}
	if doc.CurrentVersionID != nil {
		t.Error("current_version_id moved without a snapshot")
	}
}

// Clicking a parent category has to list what is filed beneath it. A document
// names exactly one category, so an exact-match filter would hide everything
// in a subtree from anyone who clicked the branch above it.
func TestCategoryFilterMatchesTheWholeSubtree(t *testing.T) {
	f := newFixture(t)

	backend, database, migrations := uuid.New(), uuid.New(), uuid.New()
	for _, c := range []struct {
		id     uuid.UUID
		parent *uuid.UUID
		name   string
	}{
		{backend, nil, "Backend"},
		{database, &backend, "Database"},
		{migrations, &database, "Migrations"},
	} {
		if err := f.db.Exec(
			"INSERT INTO categories (id, space_id, parent_id, name, slug, `order`, created_at, updated_at) VALUES (?,?,?,?,?,0,NOW(),NOW())",
			c.id, f.spaceID, c.parent, c.name, c.name+"-"+c.id.String()[:8],
		).Error; err != nil {
			t.Fatalf("failed to seed category: %v", err)
		}
	}
	t.Cleanup(func() { f.db.Exec("DELETE FROM categories WHERE space_id = ?", f.spaceID) })

	// One document per level, so a filter that matched only its own level
	// would come back with exactly one row and look plausible.
	f.db.Exec("UPDATE documents SET category_id = ? WHERE id = ?", backend, f.open)
	f.db.Exec("UPDATE documents SET category_id = ? WHERE id = ?", migrations, f.locked)

	list := func(categoryID uuid.UUID) int {
		t.Helper()
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/?category_id="+categoryID.String(), nil)

		docs, _, err := f.repo.GetAllPaginated(c, f.spaceID.String(), nil)
		if err != nil {
			t.Fatal(err)
		}
		return len(docs)
	}

	if got := list(backend); got != 2 {
		t.Errorf("the top category listed %d documents, want 2 (its own plus the one three levels down)", got)
	}
	if got := list(database); got != 1 {
		t.Errorf("the middle category listed %d, want 1 (only the leaf below it)", got)
	}
	if got := list(migrations); got != 1 {
		t.Errorf("the leaf listed %d, want 1", got)
	}
}

// RelatedTo ranks on two computed columns and filters on a HAVING, which is
// the one query in this package that gorm can silently render wrong: an ORDER
// BY carrying bind variables is dropped without an error, so the ranking has
// to travel through a select alias to be applied at all.
func TestRelatedToRanksSharedTagsAboveSharedCategory(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	categoryID := uuid.New()
	if err := f.db.Exec(
		"INSERT INTO categories (id, space_id, name, slug, `order`, created_at, updated_at) VALUES (?,?,?,?,0,NOW(),NOW())",
		categoryID, f.spaceID, "cat-"+categoryID.String()[:8], "cat-"+categoryID.String()[:8],
	).Error; err != nil {
		t.Skipf("could not seed a category: %v", err)
	}
	t.Cleanup(func() { f.db.Exec("DELETE FROM categories WHERE id = ?", categoryID) })

	tagID := uuid.New()
	if err := f.db.Exec(
		"INSERT INTO tags (id, name, slug, created_at, updated_at) VALUES (?,?,?,NOW(),NOW())",
		tagID, "tag-"+tagID.String()[:8], "tag-"+tagID.String()[:8],
	).Error; err != nil {
		t.Skipf("could not seed a tag: %v", err)
	}
	t.Cleanup(func() { f.db.Exec("DELETE FROM tags WHERE id = ?", tagID) })

	// The source, plus two candidates: one sharing its tag, one only sharing
	// its category. Everything is published — a draft is not a suggestion.
	source, byTag, byCategory, unrelated := f.open, f.locked, uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{byCategory, unrelated} {
		if err := f.db.Exec(
			`INSERT INTO documents (id, space_id, category_id, title, slug, content, status, author_id, created_at, updated_at)
			 VALUES (?,?,?,?,?,'{}','published',?,NOW(),NOW())`,
			id, f.spaceID, nil, "rel-"+id.String()[:8], "rel-"+id.String()[:8], f.authorID,
		).Error; err != nil {
			t.Fatalf("failed to seed candidate: %v", err)
		}
	}
	f.db.Exec("UPDATE documents SET status = 'published' WHERE id IN ?", []uuid.UUID{source, byTag})
	f.db.Exec("UPDATE documents SET category_id = ? WHERE id IN ?", categoryID, []uuid.UUID{source, byCategory})
	f.db.Exec("INSERT INTO document_tags (document_id, tag_id) VALUES (?,?),(?,?)", source, tagID, byTag, tagID)
	t.Cleanup(func() { f.db.Exec("DELETE FROM document_tags WHERE tag_id = ?", tagID) })

	doc, err := f.repo.GetById(ctx, source.String())
	if err != nil {
		t.Fatal(err)
	}

	related, err := f.repo.RelatedTo(ctx, doc, []uuid.UUID{tagID}, nil, RelatedLimit)
	if err != nil {
		t.Fatal(err)
	}

	if len(related) != 2 {
		t.Fatalf("expected the tag match and the category match, got %d", len(related))
	}
	if related[0].ID != byTag {
		t.Errorf("the shared tag should rank first, got %v", related[0].ID)
	}
	if related[1].ID != byCategory {
		t.Errorf("the shared category should rank second, got %v", related[1].ID)
	}
	for _, r := range related {
		if r.ID == unrelated {
			t.Error("a document sharing neither tag nor category was suggested")
		}
	}
}
