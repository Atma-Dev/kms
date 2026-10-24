package document

import (
	"api-kms/app/protection"
	"api-kms/app/user"
	"api-kms/errors"
	"api-kms/utils"
	"context"
	stderrors "errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// sortable is the allowlist behind ?sort=. The column names reach the ORDER BY
// clause as raw SQL, so a caller-supplied name must never get there — an
// unknown value falls back to the default rather than erroring, since a stale
// bookmark should still render the list.
var sortable = map[string]string{
	"updated_at": "updated_at",
	"view_count": "view_count",
	"title":      "title",
	"created_at": "created_at",
}

type Repository struct {
	db *gorm.DB
}

// GetAllPaginated lists documents in one space. Pinned documents sort first
// whatever the chosen order — that is what pinning is for.
func (r *Repository) GetAllPaginated(c *gin.Context, spaceID string, hidden []uuid.UUID) ([]Document, *utils.PaginationMeta, error) {
	column, ok := sortable[c.Query("sort")]
	if !ok {
		column = "updated_at"
	}
	direction := "DESC"
	if c.Query("direction") == "asc" {
		direction = "ASC"
	}

	db := r.db.WithContext(c).
		Where("space_id = ?", spaceID).
		Order("is_pinned DESC").
		Order(column + " " + direction)

	// Restricted documents this caller is not named on never reach the list.
	// Filtering here rather than after paginating keeps the totals honest — a
	// page of 10 that renders 7 is a bug report waiting to happen.
	if len(hidden) > 0 {
		db = db.Where("id NOT IN ?", hidden)
	}

	if status := c.Query("status"); status != "" {
		db = db.Where("status = ?", status)
	}
	if categoryID := c.Query("category_id"); categoryID != "" {
		db = InCategorySubtree(db, categoryID)
	}
	if authorID := c.Query("author_id"); authorID != "" {
		db = db.Where("author_id = ?", authorID)
	}
	if search := c.Query("search"); search != "" {
		// Title and excerpt only. Searching the content column is Modul 3's
		// job — a LIKE over a JSON blob would scan every row and still miss
		// text split across marks.
		db = db.Where("title ILIKE ? OR excerpt ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var documents []Document
	meta, err := utils.Paginate(c, db, &documents)
	return documents, meta, err
}

func (r *Repository) GetById(ctx context.Context, id string) (*Document, error) {
	// The space is joined for its protection_level alone (Modul 4 FR-6), so
	// resolving how the document must be rendered costs nothing beyond the
	// read that was already happening. Both sides of the join carry an `id`,
	// so the predicate names the table.
	var doc Document
	if err := r.db.WithContext(ctx).
		Select("documents.*, spaces.protection_level AS space_protection_level").
		Joins("JOIN spaces ON spaces.id = documents.space_id").
		First(&doc, "documents.id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("document not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get document by id").WithCause(err)
	}
	return &doc, nil
}

func (r *Repository) SlugExists(ctx context.Context, spaceID, slug, excludeID string) (bool, error) {
	// Unscoped: a soft-deleted document still owns its slug, because the
	// unique index still holds its row. Letting a new document take the slug
	// would fail at the index with a confusing message.
	db := r.db.WithContext(ctx).Unscoped().Model(&Document{}).
		Where("space_id = ? AND slug = ?", spaceID, slug)
	if excludeID != "" {
		db = db.Where("id <> ?", excludeID)
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, errors.InternalServerError("failed to check document slug").WithCause(err)
	}
	return count > 0, nil
}

func (r *Repository) AddNewData(ctx context.Context, doc *Document) error {
	if err := r.db.WithContext(ctx).Create(doc).Error; err != nil {
		return translateDuplicateSlug(err, "failed to create document")
	}
	return nil
}

// Save writes the document as it stands. Used by the lifecycle actions, which
// change status and nothing that needs a snapshot.
func (r *Repository) Save(ctx context.Context, doc *Document) error {
	if err := r.db.WithContext(ctx).Save(doc).Error; err != nil {
		return translateDuplicateSlug(err, "failed to update document")
	}
	return nil
}

// SaveWithSnapshot writes the document and, when snapshot is non-nil, records
// the pre-edit content as a version — in one transaction.
//
// The two halves cannot be separate calls (NFR: versioning consistency). A
// snapshot that commits without the edit invents a version nobody wrote; an
// edit that commits without the snapshot loses the content it replaced, which
// is the one thing versioning exists to prevent.
func (r *Repository) SaveWithSnapshot(ctx context.Context, doc *Document, snapshot *Version) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if snapshot != nil {
			number, err := nextVersionNumber(tx, doc.ID)
			if err != nil {
				return err
			}
			snapshot.VersionNumber = number

			if err := tx.Create(snapshot).Error; err != nil {
				if stderrors.Is(err, gorm.ErrDuplicatedKey) {
					// The unique key on (document_id, version_number) caught a
					// concurrent second edit. Rolling back is the correct
					// answer: the loser re-reads and retries against content
					// that now includes the winner's change.
					return errors.InvalidStatusException("this document was edited by someone else, please reload and try again").WithCause(err)
				}
				return errors.InternalServerError("failed to snapshot document version").WithCause(err)
			}

			doc.CurrentVersionID = &snapshot.ID
		}

		if err := tx.Save(doc).Error; err != nil {
			return translateDuplicateSlug(err, "failed to update document")
		}
		return nil
	})
}

func nextVersionNumber(tx *gorm.DB, documentID uuid.UUID) (uint, error) {
	var next uint
	if err := tx.Model(&Version{}).
		Where("document_id = ?", documentID).
		Select("COALESCE(MAX(version_number) + 1, 1)").
		Scan(&next).Error; err != nil {
		return 0, errors.InternalServerError("failed to get next version number").WithCause(err)
	}
	return next, nil
}

func (r *Repository) GetVersions(ctx context.Context, documentID string) ([]Version, error) {
	var versions []Version
	if err := r.db.WithContext(ctx).
		// Content is excluded: the list is a sidebar of numbers and notes, and
		// shipping every snapshot's full tree to render it would be the
		// heaviest response in the app.
		Select("id", "document_id", "version_number", "editor_id", "change_note", "created_at").
		Where("document_id = ?", documentID).
		Order("version_number DESC").
		Find(&versions).Error; err != nil {
		return nil, errors.InternalServerError("failed to get document versions").WithCause(err)
	}
	return versions, nil
}

// GetVersion loads one snapshot with its content, by version number rather
// than id: "v3" is what the UI and the compare query both name.
func (r *Repository) GetVersion(ctx context.Context, documentID string, number uint) (*Version, error) {
	var version Version
	if err := r.db.WithContext(ctx).
		First(&version, "document_id = ? AND version_number = ?", documentID, number).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("document version not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get document version").WithCause(err)
	}
	return &version, nil
}

// SoftDeleteById keeps the row (FR-3): the audit trail points at it, and a
// deleted document that vanished would leave those entries naming nothing.
func (r *Repository) SoftDeleteById(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&Document{}, "id = ?", id).Error; err != nil {
		return errors.InternalServerError("failed to delete document").WithCause(err)
	}
	return nil
}

// IncrementViewCount is a bare UPDATE rather than a read-modify-write, so two
// concurrent readers cannot both write the same number back.
func (r *Repository) IncrementViewCount(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Model(&Document{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error; err != nil {
		return errors.InternalServerError("failed to increment view count").WithCause(err)
	}
	return nil
}

func translateDuplicateSlug(err error, fallbackMessage string) error {
	if stderrors.Is(err, gorm.ErrDuplicatedKey) {
		return errors.DataAlreadyExistException("document slug already exists").
			WithErrors(map[string][]string{"title": {"a document with this title already exists in this space"}}).
			WithCause(err)
	}
	return errors.InternalServerError(fallbackMessage).WithCause(err)
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GrantsFor loads one document's access rows. Empty means unrestricted — see
// the Access doc comment for why that is the default rather than "nobody".
func (r *Repository) GrantsFor(ctx context.Context, documentID string) ([]Access, error) {
	var grants []Access
	if err := r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Order("subject_type ASC, created_at ASC").
		Find(&grants).Error; err != nil {
		return nil, errors.InternalServerError("failed to get document access").WithCause(err)
	}
	return grants, nil
}

func (r *Repository) AddGrant(ctx context.Context, grant *Access) error {
	if err := r.db.WithContext(ctx).Create(grant).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.DataAlreadyExistException("this subject already has a grant on this document").
				WithErrors(map[string][]string{"subject_id": {"already granted"}}).
				WithCause(err)
		}
		return errors.InternalServerError("failed to grant document access").WithCause(err)
	}
	return nil
}

func (r *Repository) DeleteGrant(ctx context.Context, documentID, accessID string) error {
	// Scoped by document_id as well as id: without it, an access id from
	// another document would delete a grant the caller was never shown.
	result := r.db.WithContext(ctx).
		Where("id = ? AND document_id = ?", accessID, documentID).
		Delete(&Access{})
	if result.Error != nil {
		return errors.InternalServerError("failed to revoke document access").WithCause(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.EmptyDataException("access grant not found")
	}
	return nil
}

// RestrictedIDsExcluding lists the documents in a space that are restricted
// and do NOT name this user — the set to hide from their list.
//
// One query for the whole page rather than a grants lookup per row: a space
// with a handful of restricted documents would otherwise cost one round trip
// per document on every list request.
func (r *Repository) RestrictedIDsExcluding(ctx context.Context, spaceID string, u *user.User) ([]uuid.UUID, error) {
	subjectIDs := make([]uuid.UUID, 0, len(u.Roles)+1)
	subjectIDs = append(subjectIDs, u.ID)
	for _, role := range u.Roles {
		subjectIDs = append(subjectIDs, role.ID)
	}

	db := r.db.WithContext(ctx).
		Table("document_access").
		Select("DISTINCT document_access.document_id").
		Joins("JOIN documents ON documents.id = document_access.document_id").
		// "restricted, and I am not among the subjects" — the NOT EXISTS is
		// what turns "has any grant" into "has no grant naming me".
		Where(`NOT EXISTS (
			SELECT 1 FROM document_access da
			WHERE da.document_id = document_access.document_id AND da.subject_id IN ?
		)`, subjectIDs)

	// An empty space means "every space" — search (Modul 3 FR-3) hides the
	// same documents as the space list does, and the rule is the same query
	// with one filter fewer rather than a second implementation of it.
	if spaceID != "" {
		db = db.Where("documents.space_id = ?", spaceID)
	}

	var hidden []uuid.UUID
	if err := db.Pluck("document_access.document_id", &hidden).Error; err != nil {
		return nil, errors.InternalServerError("failed to filter restricted documents").WithCause(err)
	}
	return hidden, nil
}

// InCategorySubtree narrows a document query to one category and everything
// under it. The subtree, not the one category: a document names exactly one,
// so filing it under "Migrations" would otherwise hide it from anyone who
// clicked "Backend" above it.
//
// Expanded in SQL rather than by walking the tree in Go, so this package needs
// no dependency on app/category — which needs this one for its per-category
// counts, and two-way would not build.
//
// Three levels is the cap app/category enforces (MaxDepth), so the nesting
// below is the whole tree and not a recursive CTE.
func InCategorySubtree(db *gorm.DB, categoryID string) *gorm.DB {
	return db.Where(`documents.category_id IN (
		SELECT id FROM categories WHERE id = ?
		UNION SELECT id FROM categories WHERE parent_id = ?
		UNION SELECT c.id FROM categories c
		       JOIN categories p ON p.id = c.parent_id
		       WHERE p.parent_id = ?
	)`, categoryID, categoryID, categoryID)
}

// GetDeletedById finds a soft-deleted document. Unscoped, because GORM's
// default scope hides exactly the rows this is looking for.
func (r *Repository) GetDeletedById(ctx context.Context, id string) (*Document, error) {
	var doc Document
	if err := r.db.WithContext(ctx).Unscoped().
		First(&doc, "id = ? AND deleted_at IS NOT NULL", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("deleted document not found")
		}
		return nil, errors.InternalServerError("failed to get deleted document").WithCause(err)
	}
	return &doc, nil
}

// UndeleteById clears deleted_at. The slug cannot collide on the way back: a
// soft-deleted row keeps holding its slug (SlugExists is Unscoped for exactly
// this reason), so nothing else can have taken it meanwhile.
func (r *Repository) UndeleteById(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Unscoped().Model(&Document{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error; err != nil {
		return errors.InternalServerError("failed to restore document").WithCause(err)
	}
	return nil
}

// GetDeletedPaginated is the bin for one space.
func (r *Repository) GetDeletedPaginated(c *gin.Context, spaceID string) ([]Document, *utils.PaginationMeta, error) {
	db := r.db.WithContext(c).Unscoped().
		Where("space_id = ? AND deleted_at IS NOT NULL", spaceID).
		Order("deleted_at DESC")

	var documents []Document
	meta, err := utils.Paginate(c, db, &documents)
	return documents, meta, err
}

// SetPinned is a single-column write, not a Save: pinning must not carry the
// caller's copy of the content back to the database, or a pin would silently
// overwrite an edit someone else made in between.
// SetProtectionLevel writes the per-document override (Modul 4 FR-6). A nil
// level writes NULL, which is what "inherit the space" is stored as.
//
// An Update of the one column rather than a Save of the row: the document was
// loaded for its id alone, and a full-row save here would write back whatever
// a concurrent edit had already changed underneath it.
func (r *Repository) SetProtectionLevel(ctx context.Context, id string, level *protection.Level) error {
	if err := r.db.WithContext(ctx).Model(&Document{}).
		Where("id = ?", id).
		Update("protection_level", level).Error; err != nil {
		return errors.InternalServerError("failed to set the protection level").WithCause(err)
	}
	return nil
}

func (r *Repository) SetPinned(ctx context.Context, id string, pinned bool) error {
	if err := r.db.WithContext(ctx).Model(&Document{}).
		Where("id = ?", id).
		Update("is_pinned", pinned).Error; err != nil {
		return errors.InternalServerError("failed to pin document").WithCause(err)
	}
	return nil
}

// RelatedTo is FR-4: the documents a reader of this one is most likely to want
// next. Tag overlap first, same category as the tie-breaker, most recently
// updated to settle the rest.
//
// The candidate set is "shares a tag OR sits in the same category" — without
// that HAVING, a document with no tags would pull in whatever else happened to
// be in the space, which is a list of unrelated documents wearing the label
// "related".
//
// Scoped to the document's own space. Two spaces may use the same tag for
// unrelated things, and a suggestion a reader cannot open is worse than no
// suggestion.
func (r *Repository) RelatedTo(ctx context.Context, doc *Document, tagIDs, hidden []uuid.UUID, limit int) ([]Document, error) {
	// A document with no category has no category tie-breaker: the literal 0
	// keeps the ordering valid rather than sorting on NULL, which a DESC sort
	// puts first and so silently reverses the intent. The CASE makes it an
	// integer either way — a bare comparison would be a boolean, which cannot
	// be compared with the 1 below.
	sameCategory := "0"
	categoryArgs := []any{}
	if doc.CategoryID != nil {
		sameCategory = "CASE WHEN documents.category_id = ? THEN 1 ELSE 0 END"
		categoryArgs = append(categoryArgs, *doc.CategoryID)
	}

	db := r.db.WithContext(ctx).Model(&Document{}).
		// Both ranking terms are computed in the SELECT and ordered by alias.
		// gorm.Expr is not an ORDER BY the driver accepts — it is dropped
		// without an error — so an expression with bind variables has to come
		// through here to be sorted on at all.
		Select("documents.*, COUNT(dt.tag_id) AS shared_tags, ("+sameCategory+") AS same_category", categoryArgs...).
		// An empty tagIDs renders as IN (NULL): no row matches, the count is
		// zero everywhere, and the HAVING falls back to category alone.
		Joins("LEFT JOIN document_tags dt ON dt.document_id = documents.id AND dt.tag_id IN ?", tagIDs).
		Where("documents.id <> ?", doc.ID).
		Where("documents.space_id = ?", doc.SpaceID).
		// Only published: a suggestion is a recommendation to go and read
		// something, and a draft is not ready to be read.
		Where("documents.status = ?", StatusPublished).
		// documents.* under a GROUP BY on the id alone is valid because id is
		// the primary key: every other column is functionally dependent on it.
		Group("documents.id").
		// The HAVING repeats the expressions rather than naming the select
		// aliases: Postgres resolves aliases in ORDER BY but not in HAVING.
		Having("COUNT(dt.tag_id) > 0 OR ("+sameCategory+") = 1", categoryArgs...).
		Order("shared_tags DESC, same_category DESC, documents.updated_at DESC").
		Limit(limit)

	if len(hidden) > 0 {
		db = db.Where("documents.id NOT IN ?", hidden)
	}

	var related []Document
	if err := db.Find(&related).Error; err != nil {
		return nil, errors.InternalServerError("failed to get related documents").WithCause(err)
	}
	return related, nil
}

// RecordView writes the detail behind a view_count increment — who read what,
// and when. Called only where the count is incremented, under the same
// once-per-hour throttle, so the row and the number never disagree.
func (r *Repository) RecordView(ctx context.Context, documentID, userID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Table("document_views").Create(map[string]any{
		"id":          uuid.New(),
		"document_id": documentID,
		"user_id":     userID,
		"viewed_at":   time.Now(),
	}).Error; err != nil {
		return errors.InternalServerError("failed to record document view").WithCause(err)
	}
	return nil
}
