package search

import (
	"api-kms/app/user"
	"api-kms/internal/testdb"
	"api-kms/utils"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// fixture builds a throwaway space whose documents are written to make the
// ranking visible: one has the keyword in its title, the other only in its
// body. Everything it creates is removed again, so a run leaves the dev
// database exactly as it found it.
type fixture struct {
	db       *gorm.DB
	repo     *Repository
	spaceID  uuid.UUID
	authorID uuid.UUID

	titleMatch uuid.UUID // keyword in the title
	bodyMatch  uuid.UUID // keyword only in the body
	otherDraft uuid.UUID // someone else's unpublished draft
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testdb.Open(t)

	var author user.User
	if err := db.First(&author).Error; err != nil {
		t.Skipf("no users in the dev database: %v", err)
	}

	f := &fixture{
		db: db, repo: NewRepository(db),
		spaceID: uuid.New(), authorID: author.ID,
		titleMatch: uuid.New(), bodyMatch: uuid.New(), otherDraft: uuid.New(),
	}

	if err := db.Exec(
		"INSERT INTO spaces (id, name, slug, is_archived, created_by, created_at, updated_at) VALUES (?,?,?,0,?,NOW(),NOW())",
		f.spaceID, "search-"+f.spaceID.String()[:8], "search-"+f.spaceID.String()[:8], author.ID,
	).Error; err != nil {
		t.Fatalf("failed to seed space: %v", err)
	}

	f.seed(t, f.titleMatch, "Deployment Runbook", "brewing coffee for the morning standup", "published")
	f.seed(t, f.bodyMatch, "Coffee Machine", "the deployment pipeline runs nightly", "published")
	f.seed(t, f.otherDraft, "Deployment Notes", "deployment deployment deployment", "draft")

	t.Cleanup(func() {
		db.Exec("DELETE FROM documents WHERE space_id = ?", f.spaceID)
		db.Exec("DELETE FROM spaces WHERE id = ?", f.spaceID)
	})

	return f
}

func (f *fixture) seed(t *testing.T, id uuid.UUID, title, body, status string) {
	t.Helper()
	if err := f.db.Exec(
		`INSERT INTO documents (id, space_id, title, slug, content, excerpt, search_body, status, author_id, created_at, updated_at)
		 VALUES (?,?,?,?,'{}',?,?,?,?,NOW(),NOW())`,
		id, f.spaceID, title, "doc-"+id.String()[:8], body, body, status, f.authorID,
	).Error; err != nil {
		t.Fatalf("failed to seed document %q: %v", title, err)
	}
}

// searchAs runs a search scoped to this fixture's space, as the given viewer.
func (f *fixture) searchAs(t *testing.T, viewerID uuid.UUID, query string) []uuid.UUID {
	t.Helper()

	documents, _, err := f.repo.Search(context.Background(), Query{
		Text:     query,
		SpaceID:  f.spaceID.String(),
		Page:     1,
		PerPage:  utils.DefaultPerPage,
		ViewerID: viewerID,
	})
	if err != nil {
		t.Fatalf("search for %q failed: %v", query, err)
	}

	ids := make([]uuid.UUID, len(documents))
	for i := range documents {
		ids[i] = documents[i].ID
	}
	return ids
}

// The PRD's setweight('A') for the title against setweight('C') for the body:
// a document *called* "Deployment Runbook" is nearly always the one meant, and
// one that mentions deployment in a sentence rarely is.
//
// This is also the only check that the MATCH ... AGAINST expression survives
// gorm at all — the score has to reach the ORDER BY through a select alias,
// and utils.Paginate's COUNT has to stay valid once it replaces that select.
func TestSearchRanksATitleMatchAboveABodyMatch(t *testing.T) {
	f := newFixture(t)

	got := f.searchAs(t, uuid.New(), "deployment")

	if len(got) != 2 {
		t.Fatalf("expected the two published documents, got %d: %v", len(got), got)
	}
	if got[0] != f.titleMatch {
		t.Errorf("the title match should rank first, got %v", got)
	}
	if got[1] != f.bodyMatch {
		t.Errorf("the body match should rank second, got %v", got)
	}
}

// FR-3: only published documents are searchable, except the caller's own
// drafts. A draft someone else is still writing is not knowledge yet, and
// finding it by keyword is how an unfinished answer gets acted on.
func TestSearchShowsOnlyMyOwnDrafts(t *testing.T) {
	f := newFixture(t)

	for _, id := range f.searchAs(t, uuid.New(), "deployment") {
		if id == f.otherDraft {
			t.Fatal("a stranger found somebody else's draft")
		}
	}

	found := false
	for _, id := range f.searchAs(t, f.authorID, "deployment") {
		if id == f.otherDraft {
			found = true
		}
	}
	if !found {
		t.Error("the author cannot find their own draft")
	}
}

// Every word below innodb_ft_min_token_size means the full-text index holds
// nothing to match, and the query has to fall back to a LIKE rather than
// answering "no results" to a word that is plainly in the document.
func TestSearchFallsBackToLikeWhenNothingIsIndexable(t *testing.T) {
	f := newFixture(t)
	f.seed(t, uuid.New(), "CI overview", "runs on every push", "published")

	got := f.searchAs(t, uuid.New(), "CI")

	if len(got) == 0 {
		t.Fatal("a two-letter query found nothing; the LIKE fallback did not run")
	}
}

// Searching the same thing twice is one entry that moved, not two rows — which
// is also what gives DELETE /search/history/{id} something stable to name.
func TestRememberQueryUpsertsInsteadOfAppending(t *testing.T) {
	db := testdb.Open(t)
	repo := NewRepository(db)
	ctx := context.Background()

	var owner user.User
	if err := db.First(&owner).Error; err != nil {
		t.Skipf("no users in the dev database: %v", err)
	}
	query := "deployment " + uuid.New().String()[:8]
	t.Cleanup(func() { db.Exec("DELETE FROM search_histories WHERE user_id = ? AND query = ?", owner.ID, query) })

	for range 3 {
		if err := repo.RememberQuery(ctx, owner.ID, query); err != nil {
			t.Fatalf("failed to remember the query: %v", err)
		}
	}

	var rows int64
	db.Table("search_histories").Where("user_id = ? AND query = ?", owner.ID, query).Count(&rows)
	if rows != 1 {
		t.Errorf("three identical searches left %d rows, want 1", rows)
	}
}

// PruneHistories is a single statement that deletes across every account at
// once, so it is exactly the kind of query that is either right or quietly
// destructive — it has to keep the newest rows, and only for the user it is
// counting.
func TestPruneHistoriesKeepsTheNewestPerUser(t *testing.T) {
	db := testdb.Open(t)
	repo := NewRepository(db)
	ctx := context.Background()

	var users []user.User
	if err := db.Limit(2).Find(&users).Error; err != nil || len(users) < 2 {
		t.Skipf("need two users in the dev database: %v", err)
	}
	mine, theirs := users[0].ID, users[1].ID

	prefix := "prune-" + uuid.New().String()[:8] + "-"
	t.Cleanup(func() { db.Exec("DELETE FROM search_histories WHERE query LIKE ?", prefix+"%") })

	// Five of mine, oldest first, plus one of somebody else's — which must
	// survive whatever happens to mine.
	for i := range 5 {
		if err := repo.RememberQuery(ctx, mine, prefix+string(rune('a'+i))); err != nil {
			t.Fatal(err)
		}
		// updated_at is what the prune ranks on, and five upserts inside one
		// second would all carry the same value.
		db.Exec("UPDATE search_histories SET updated_at = ? WHERE user_id = ? AND query = ?",
			time.Now().Add(time.Duration(i)*time.Minute), mine, prefix+string(rune('a'+i)))
	}
	if err := repo.RememberQuery(ctx, theirs, prefix+"theirs"); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.PruneHistories(ctx, 2); err != nil {
		t.Fatal(err)
	}

	var kept []string
	db.Table("search_histories").
		Where("user_id = ? AND query LIKE ?", mine, prefix+"%").
		Order("updated_at DESC").
		Pluck("query", &kept)

	if len(kept) != 2 || kept[0] != prefix+"e" || kept[1] != prefix+"d" {
		t.Errorf("kept %v, want the two newest (%se, %sd)", kept, prefix, prefix)
	}

	var theirRows int64
	db.Table("search_histories").Where("user_id = ? AND query LIKE ?", theirs, prefix+"%").Count(&theirRows)
	if theirRows != 1 {
		t.Errorf("another user's single row was pruned along with mine (%d left)", theirRows)
	}
}

// The whole point of reading the token size from the server is that it is not
// assumed, so the check has to be "it matches what MySQL says" — asserting it
// equals 3 would pass just as well if the read had been quietly dropped.
func TestNewRepositoryReadsTheServersMinimumTokenSize(t *testing.T) {
	db := testdb.Open(t)

	var want int
	if err := db.Raw("SELECT @@innodb_ft_min_token_size").Scan(&want).Error; err != nil {
		t.Skipf("server will not report innodb_ft_min_token_size: %v", err)
	}

	if got := NewRepository(db).minToken; got != want {
		t.Errorf("minToken = %d, want the server's %d", got, want)
	}
}
