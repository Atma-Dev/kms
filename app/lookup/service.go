package lookup

import (
	"api-kms/app/category"
	"api-kms/app/rbac"
	"api-kms/app/space"
	"api-kms/app/tag"
	"api-kms/app/user"
	"api-kms/errors"
	"context"
	"strings"
)

// Resource names the lookup a request is asking for. They are the URL segment
// of /api/lookups/:resource, and each maps to exactly one lookups.* permission
// in routes/route.go.
const (
	ResourceUsers       = "users"
	ResourceRoles       = "roles"
	ResourcePermissions = "permissions"
	ResourceSpaces      = "spaces"
	ResourceCategories  = "categories"
	ResourceTags        = "tags"
)

type Service struct {
	userRepository       *user.Repository
	roleRepository       *rbac.RoleRepository
	permissionRepository *rbac.PermissionRepository
	spaceRepository      *space.Repository
	categoryRepository   *category.Repository
	tagRepository        *tag.Repository
}

// Get returns one lookup's options. The resource has already been permission
// checked by the route's RequirePermission — this only fetches. An unknown
// name still 404s rather than falling through to an empty list, so a typo in a
// caller is visible instead of silently rendering an empty dropdown.
func (s *Service) Get(ctx context.Context, resource, scope string) ([]Option, error) {
	switch resource {
	case ResourceUsers:
		users, err := s.userRepository.GetLookup(ctx)
		if err != nil {
			return nil, err
		}
		return usersToOptions(users), nil

	case ResourceRoles:
		roles, err := s.roleRepository.GetAll(ctx)
		if err != nil {
			return nil, err
		}
		options := make([]Option, 0, len(roles))
		for _, role := range roles {
			options = append(options, Option{ID: role.ID.String(), Name: role.Name})
		}
		return options, nil

	case ResourceSpaces:
		spaces, err := s.spaceRepository.GetLookup(ctx)
		if err != nil {
			return nil, err
		}
		options := make([]Option, 0, len(spaces))
		for i := range spaces {
			options = append(options, Option{ID: spaces[i].ID.String(), Name: spaces[i].Name})
		}
		return options, nil

	case ResourceCategories:
		// Scoped by ?space_id=, which the handler passes through — a category
		// dropdown is only ever filled for one space at a time, and returning
		// every space's categories would let the caller file a document under
		// a category that does not belong to it.
		rows, err := s.categoryRepository.GetBySpace(ctx, scope)
		if err != nil {
			return nil, err
		}
		// Nested, then flattened back with each option carrying its full path:
		// a flat list of bare names cannot tell two branches' "Migrations"
		// apart, and the tree order is what the picker should read in.
		pathed := category.FlattenPaths(category.BuildTree(rows))
		options := make([]Option, 0, len(pathed))
		for i := range pathed {
			options = append(options, Option{ID: pathed[i].ID.String(), Name: pathed[i].Path})
		}
		return options, nil

	case ResourceTags:
		tags, err := s.tagRepository.GetAll(ctx, "")
		if err != nil {
			return nil, err
		}
		options := make([]Option, 0, len(tags))
		for i := range tags {
			options = append(options, Option{ID: tags[i].ID.String(), Name: tags[i].Name})
		}
		return options, nil

	case ResourcePermissions:
		permissions, err := s.permissionRepository.GetAll(ctx)
		if err != nil {
			return nil, err
		}
		options := make([]Option, 0, len(permissions))
		for _, permission := range permissions {
			options = append(options, Option{ID: permission.ID.String(), Name: permission.Name})
		}
		return options, nil
	}

	return nil, errors.EmptyDataException("lookup resource not found")
}

// usersToOptions labels a user the same way UserResource does, so a name in a
// dropdown matches the name in the table it filters.
func usersToOptions(users []user.User) []Option {
	options := make([]Option, 0, len(users))
	for _, u := range users {
		name := strings.TrimSpace(u.FirstName + " " + derefString(u.LastName))
		if name == "" {
			name = u.Email
		}
		options = append(options, Option{ID: u.ID.String(), Name: name})
	}
	return options
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func NewService(
	userRepository *user.Repository,
	roleRepository *rbac.RoleRepository,
	permissionRepository *rbac.PermissionRepository,
	spaceRepository *space.Repository,
	categoryRepository *category.Repository,
	tagRepository *tag.Repository,
) *Service {
	return &Service{
		userRepository:       userRepository,
		roleRepository:       roleRepository,
		permissionRepository: permissionRepository,
		spaceRepository:      spaceRepository,
		categoryRepository:   categoryRepository,
		tagRepository:        tagRepository,
	}
}
