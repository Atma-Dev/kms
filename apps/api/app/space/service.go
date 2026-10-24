package space

import (
	"api-kms/app/protection"
	"api-kms/errors"
	"api-kms/utils"
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goaudit "github.com/iqbalatma/go-audit"
)

type Service struct {
	repository *Repository
}

func (s *Service) GetAllPaginated(c *gin.Context) ([]Space, *utils.PaginationMeta, error) {
	return s.repository.GetAllPaginated(c)
}

func (s *Service) GetById(ctx context.Context, id string) (*Space, error) {
	return s.repository.GetById(ctx, id)
}

func (s *Service) AddNew(ctx context.Context, request *StoreSpaceRequest, createdBy uuid.UUID) (*Space, error) {
	slug, err := s.uniqueSlug(ctx, request.Name, "")
	if err != nil {
		return nil, err
	}

	space := Space{
		Name:        request.Name,
		Slug:        slug,
		Description: request.Description,
		Icon:        request.Icon,
		CreatedBy:   createdBy,
	}
	if err := s.repository.AddNewData(ctx, &space); err != nil {
		return nil, err
	}

	utils.Forget(utils.ReadCachePrefix)

	goaudit.Init(ctx, "CREATE_SPACE", "create space via space.Service.AddNew").
		SetEntryObject("spaces", space.ID.String()).
		AddSingleTrail("spaces", space.ID.String(), nil, map[string]any{
			"name": space.Name, "slug": space.Slug, "icon": space.Icon,
		}, nil, nil).
		Execute(ctx)

	return &space, nil
}

func (s *Service) UpdateById(ctx context.Context, id string, request *UpdateSpaceRequest) (*Space, error) {
	space, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	before := map[string]any{
		"name": space.Name, "slug": space.Slug, "description": space.Description,
		"icon": space.Icon, "is_archived": space.IsArchived,
	}

	// The slug only moves when the name does. A space that is merely being
	// re-described keeps its URL — links to it are already out in the world.
	if request.Name != space.Name {
		slug, err := s.uniqueSlug(ctx, request.Name, space.ID.String())
		if err != nil {
			return nil, err
		}
		space.Slug = slug
	}

	space.Name = request.Name
	space.Description = request.Description
	space.Icon = request.Icon
	if request.IsArchived != nil {
		space.IsArchived = *request.IsArchived
	}

	if err := s.repository.UpdateById(ctx, space); err != nil {
		return nil, err
	}

	utils.Forget(utils.ReadCachePrefix)

	goaudit.Init(ctx, "UPDATE_SPACE", "update space via space.Service.UpdateById").
		SetEntryObject("spaces", space.ID.String()).
		AddSingleTrail("spaces", space.ID.String(), before, map[string]any{
			"name": space.Name, "slug": space.Slug, "description": space.Description,
			"icon": space.Icon, "is_archived": space.IsArchived,
		}, nil, nil).
		Execute(ctx)

	return space, nil
}

// Archive is what DELETE /spaces/{id} does. A space owns documents, so
// removing the row would orphan or cascade-destroy every one of them — see the
// table comment in the migration.
func (s *Service) Archive(ctx context.Context, id string) (*Space, error) {
	space, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}
	if space.IsArchived {
		return space, nil
	}

	space.IsArchived = true
	if err := s.repository.UpdateById(ctx, space); err != nil {
		return nil, err
	}

	utils.Forget(utils.ReadCachePrefix)

	goaudit.Init(ctx, "ARCHIVE_SPACE", "archive space via space.Service.Archive").
		SetEntryObject("spaces", space.ID.String()).
		AddSingleTrail("spaces", space.ID.String(),
			map[string]any{"is_archived": false}, map[string]any{"is_archived": true}, nil, nil).
		Execute(ctx)

	return space, nil
}

// SetProtectionLevel writes the space default (Modul 4 FR-6).
//
// A space has nothing to inherit from, so its level is required rather than
// nullable — the nullable side of the pair is the per-document override, which
// is where "inherit" actually means something.
func (s *Service) SetProtectionLevel(ctx context.Context, id string, level protection.Level) (*Space, error) {
	if !level.Valid() {
		return nil, errors.InvalidStatusException("unknown protection level").
			WithErrors(map[string][]string{"protection_level": {"must be normal or strict"}})
	}

	space, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	before := space.ProtectionLevel
	space.ProtectionLevel = level
	if err := s.repository.UpdateById(ctx, space); err != nil {
		return nil, err
	}

	utils.Forget(utils.ReadCachePrefix)

	goaudit.Init(ctx, "SET_SPACE_PROTECTION", "set space protection level via space.Service.SetProtectionLevel").
		SetEntryObject("spaces", space.ID.String()).
		AddSingleTrail("spaces", space.ID.String(),
			map[string]any{"protection_level": string(before)},
			map[string]any{"protection_level": string(level)}, nil, nil).
		Execute(ctx)

	return space, nil
}

func (s *Service) uniqueSlug(ctx context.Context, name, excludeID string) (string, error) {
	return utils.UniqueSlug(utils.Slugify(name), func(candidate string) (bool, error) {
		return s.repository.SlugExists(ctx, candidate, excludeID)
	})
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}
