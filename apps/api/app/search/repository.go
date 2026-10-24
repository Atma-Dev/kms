package search

import (
	"api-kms/app/document"
	"api-kms/errors"
	"api-kms/utils"
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SuggestLimit is how many titles the autocomplete offers (FR-6). Long enough
// to contain the right one, short enough to read without scrolling.
const SuggestLimit = 8

// HistoryLimit is how many past queries the search box shows when it is
// focused and empty (FR-5).
const HistoryLimit = 10

// likeEscaper neutralises the wildcards in a literal the caller typed. Without
// it a search for "50%" matches every document, which reads as a bug rather
// than as a feature nobody asked for.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Repository is the MySQL Engine, and also the store behind the search
// history — which is not part of Engine. See engine.go.
type Repository struct {
	db *gorm.DB
	// minToken is the server's innodb_ft_min_token_size, read once at
	// construction. Words shorter than it are not in the index at all, which
	// is what decides whether a query can use the index or has to fall back
	// to a LIKE.
	minToken int
}

// Search is FR-1 through FR-3 in one query.
//
// The ranking expression is computed in the SELECT and ordered on by alias.
// That is not cosmetic: gorm drops an ORDER BY carrying bind variables without
// reporting it, and utils.PaginateAt's COUNT replaces the SELECT clause and
// its variables wholesale — so this is the one shape where the score reaches
// the sort and the count query stays valid.
func (r *Repository) Search(ctx context.Context, q Query) ([]document.Document, *utils.PaginationMeta, error) {
	db := r.db.WithContext(ctx).Model(&document.Document{})

	// FR-3, first half: published, or a draft the caller wrote. The author
	// clause is not a permission check — it is the PRD's "kecuali author
	// mencari draft miliknya sendiri", and it is deliberately not widened to
	// reviewers: a reviewer's queue is a list, not a search result.
	db = db.Where("documents.status = ? OR documents.author_id = ?", document.StatusPublished, q.ViewerID)

	// FR-3, second half: the restricted documents this caller is not named on,
	// resolved by app/document so search hides exactly what the space list
	// hides.
	if len(q.Hidden) > 0 {
		db = db.Where("documents.id NOT IN ?", q.Hidden)
	}

	scored := false
	if q.Text != "" {
		if expression := booleanQuery(q.Text, r.minToken); expression != "" {
			scored = true
			db = db.
				Where("MATCH(documents.title) AGAINST (? IN BOOLEAN MODE) OR MATCH(documents.search_body) AGAINST (? IN BOOLEAN MODE)", expression, expression).
				Select("documents.*, (MATCH(documents.title) AGAINST (? IN BOOLEAN MODE) * ? + MATCH(documents.search_body) AGAINST (? IN BOOLEAN MODE)) AS score",
					expression, TitleWeight, expression)
		} else {
			// Every word was shorter than the index's minimum token, so the
			// full-text index holds none of them. A LIKE over title and
			// excerpt is slow and unranked, and it is still the difference
			// between finding "CI" and being told it does not exist.
			like := "%" + likeEscaper.Replace(q.Text) + "%"
			db = db.Where("documents.title LIKE ? OR documents.excerpt LIKE ?", like, like)
		}
	}

	db = applyFilters(db, q)

	// FR-2. Relevance is only meaningful when something was matched against —
	// browsing the filters with an empty box sorts by newest instead, which is
	// what an unranked list should do.
	switch q.Sort {
	case SortNewest:
		db = db.Order("documents.updated_at DESC")
	case SortPopular:
		db = db.Order("documents.view_count DESC")
	default:
		if scored {
			db = db.Order("score DESC, documents.updated_at DESC")
		} else {
			db = db.Order("documents.updated_at DESC")
		}
	}

	var documents []document.Document
	meta, err := utils.PaginateAt(db, q.Page, q.PerPage, q.Path, &documents)
	return documents, meta, err
}

// applyFilters is FR-2's filter half. It lives apart from Search only because
// the ranking above is hard enough to read without six more clauses in the
// middle of it.
func applyFilters(db *gorm.DB, q Query) *gorm.DB {
	if q.SpaceID != "" {
		db = db.Where("documents.space_id = ?", q.SpaceID)
	}
	if q.CategoryID != "" {
		// The subtree, same as the space's document list — app/document owns
		// that expansion and this asks it rather than copying it.
		db = document.InCategorySubtree(db, q.CategoryID)
	}
	if q.AuthorID != "" {
		db = db.Where("documents.author_id = ?", q.AuthorID)
	}
	if q.TagID != "" {
		// A subquery rather than a join: joining document_tags multiplies a
		// document by its tags, and a page of 10 would render the same
		// document three times because it carries three tags.
		db = db.Where("documents.id IN (SELECT dt.document_id FROM document_tags dt WHERE dt.tag_id = ?)", q.TagID)
	}
	if q.From != nil {
		db = db.Where("documents.updated_at >= ?", *q.From)
	}
	if q.To != nil {
		db = db.Where("documents.updated_at < ?", *q.To)
	}
	return db
}

// Suggest is FR-6: titles that start with what has been typed so far.
//
// A prefix LIKE, not the full-text index. The index matches words anywhere in
// a title, which is the right answer for a search and the wrong one for a
// dropdown that is supposed to be completing the thing being typed.
func (r *Repository) Suggest(ctx context.Context, q Query) ([]document.Document, error) {
	db := r.db.WithContext(ctx).Model(&document.Document{}).
		Select("documents.id", "documents.space_id", "documents.title", "documents.slug", "documents.status").
		Where("documents.title LIKE ?", likeEscaper.Replace(q.Text)+"%").
		Where("documents.status = ? OR documents.author_id = ?", document.StatusPublished, q.ViewerID).
		// Most read first: when two titles both fit, the one people actually
		// open is the one being reached for.
		Order("documents.view_count DESC").
		Limit(SuggestLimit)

	if q.SpaceID != "" {
		db = db.Where("documents.space_id = ?", q.SpaceID)
	}
	if len(q.Hidden) > 0 {
		db = db.Where("documents.id NOT IN ?", q.Hidden)
	}

	var documents []document.Document
	if err := db.Find(&documents).Error; err != nil {
		return nil, errors.InternalServerError("failed to get search suggestions").WithCause(err)
	}
	return documents, nil
}

// RememberQuery is FR-5's write. An upsert, so searching the same thing twice
// moves one row to the top instead of filling the list with itself — and so
// two searches racing cannot both insert, which the unique key would reject.
func (r *Repository) RememberQuery(ctx context.Context, userID uuid.UUID, query string) error {
	entry := History{UserID: userID, Query: query}

	// updated_at is what moves, not created_at: the list is ordered by "last
	// searched" and created_at keeps meaning "first searched", which is the
	// honest reading of both columns.
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "query"}},
		DoUpdates: clause.Assignments(map[string]any{"updated_at": time.Now()}),
	}).Create(&entry).Error; err != nil {
		return errors.InternalServerError("failed to save search history").WithCause(err)
	}
	return nil
}

func (r *Repository) Histories(ctx context.Context, userID uuid.UUID) ([]History, error) {
	var histories []History
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(HistoryLimit).
		Find(&histories).Error; err != nil {
		return nil, errors.InternalServerError("failed to get search history").WithCause(err)
	}
	return histories, nil
}

// DeleteHistory is scoped by user as well as id: without it, an id from
// somebody else's history would delete a row this caller was never shown.
func (r *Repository) DeleteHistory(ctx context.Context, userID uuid.UUID, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&History{})
	if result.Error != nil {
		return errors.InternalServerError("failed to delete search history").WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.EmptyDataException("search history entry not found")
	}
	return nil
}

// PruneHistories drops everything past the newest keep queries per user.
//
// One statement, not a read-then-delete per user: the whole point is that this
// runs unattended over a table nobody is watching, and a sweep that costs one
// round trip per account is the kind of job that quietly stops finishing.
//
// ROW_NUMBER rather than a correlated subquery for the same reason — MySQL 8
// ranks every user's rows in one pass instead of re-scanning the table once
// per row.
func (r *Repository) PruneHistories(ctx context.Context, keep int) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`
		DELETE sh FROM search_histories sh
		JOIN (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY updated_at DESC) AS rn
			FROM search_histories
		) ranked ON ranked.id = sh.id
		WHERE ranked.rn > ?`, keep)
	if result.Error != nil {
		return 0, errors.InternalServerError("failed to prune search history").WithCause(result.Error)
	}
	return result.RowsAffected, nil
}

// NewRepository reads the server's minimum full-text token size once, here,
// rather than on every query: it is a server variable that cannot change
// without a restart of MySQL, and a SELECT per search to learn a constant is
// the wrong trade in both directions.
//
// A server that will not answer keeps the InnoDB default. Refusing to start
// over an optimisation hint would be worse than being slightly wrong about
// which two-letter words need the LIKE fallback.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db, minToken: readMinTokenSize(db)}
}

func readMinTokenSize(db *gorm.DB) int {
	var size int
	if err := db.Raw("SELECT @@innodb_ft_min_token_size").Scan(&size).Error; err != nil || size < 1 {
		return DefaultMinTokenSize
	}
	return size
}
