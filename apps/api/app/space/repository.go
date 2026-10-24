package space

import (
	"api-kms/errors"
	"api-kms/utils"
	"context"
	stderrors "errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func (r *Repository) GetAllPaginated(c *gin.Context) ([]Space, *utils.PaginationMeta, error) {
	db := r.db.WithContext(c).Order("name ASC")

	if search := c.Query("search"); search != "" {
		db = db.Where("name ILIKE ?", "%"+search+"%")
	}
	// Archived spaces are hidden unless asked for: the list is mostly used to
	// pick where a document goes, and an archived space is not a valid answer.
	if c.Query("include_archived") != "1" {
		db = db.Where("is_archived = ?", false)
	}

	var spaces []Space
	meta, err := utils.Paginate(c, db, &spaces)
	return spaces, meta, err
}

// GetLookup backs the space dropdown: active spaces only, cheapest columns.
func (r *Repository) GetLookup(ctx context.Context) ([]Space, error) {
	return utils.Remember(utils.ReadCachePrefix+"spaces:lookup", utils.OwnTableCacheTTL, func() ([]Space, error) {
		var spaces []Space
		if err := r.db.WithContext(ctx).
			Select("id", "name", "slug").
			Where("is_archived = ?", false).
			Order("name ASC").
			Find(&spaces).Error; err != nil {
			return nil, errors.InternalServerError("failed to get space lookup").WithCause(err)
		}
		return spaces, nil
	})
}

func (r *Repository) GetById(ctx context.Context, id string) (*Space, error) {
	var s Space
	if err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("space not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get space by id").WithCause(err)
	}
	return &s, nil
}

// SlugExists reports whether a slug is taken, ignoring the row being updated —
// without excludeID a rename that keeps the slug would collide with itself.
func (r *Repository) SlugExists(ctx context.Context, slug, excludeID string) (bool, error) {
	db := r.db.WithContext(ctx).Model(&Space{}).Where("slug = ?", slug)
	if excludeID != "" {
		db = db.Where("id <> ?", excludeID)
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, errors.InternalServerError("failed to check space slug").WithCause(err)
	}
	return count > 0, nil
}

func (r *Repository) AddNewData(ctx context.Context, s *Space) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return translateDuplicateSlug(err, "failed to create space")
	}
	return nil
}

func (r *Repository) UpdateById(ctx context.Context, s *Space) error {
	if err := r.db.WithContext(ctx).Save(s).Error; err != nil {
		return translateDuplicateSlug(err, "failed to update space")
	}
	return nil
}

// translateDuplicateSlug turns the unique index violation into a 409 naming the
// field, rather than the generic 500 a raw driver error would get. UniqueSlug
// already avoids this in the normal case — this is the concurrent one.
func translateDuplicateSlug(err error, fallbackMessage string) error {
	if stderrors.Is(err, gorm.ErrDuplicatedKey) {
		return errors.DataAlreadyExistException("space slug already exists").
			WithErrors(map[string][]string{"name": {"a space with this name already exists"}}).
			WithCause(err)
	}
	return errors.InternalServerError(fallbackMessage).WithCause(err)
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}
