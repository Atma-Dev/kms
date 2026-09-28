package rbac

import (
	"context"
	"api-kms/config"
	"api-kms/errors"
	"api-kms/utils"
	"sync"

	"github.com/google/uuid"
	goaudit "github.com/iqbalatma/go-audit"
)

// permissionNames extracts each permission's slug for an audit trail — the
// names are the meaningful, readable identifier here, not the row ids.
func permissionNames(permissions []Permission) []string {
	names := make([]string, len(permissions))
	for i, p := range permissions {
		names[i] = p.Name
	}
	return names
}

type Service struct {
	permissionRepository *PermissionRepository
	roleRepository       *RoleRepository
	// Pushes a control frame to the users a role change affects. Their
	// frontend caches the permission list, so without this it keeps drawing
	// the old menu until a full page load — the API itself is never stale,
	// AuthMiddleware re-reads Roles.Permissions on every request.
	//
	// An interface, not *notification.Hub: that package imports user, which
	// imports this one, so naming the concrete type here would cycle.
	broadcaster PermissionBroadcaster
}

// PermissionBroadcaster is satisfied by *notification.Hub.
type PermissionBroadcaster interface {
	BroadcastPermissionsChanged(userIDs []string)
}

// announcePermissionChange tells everyone holding roleID to re-read their
// permissions. Deliberately after the write has committed, and deliberately
// not part of it: a websocket push is best effort, and failing to deliver it
// must not roll back a role the admin already saved. The cost of a missed
// frame is one stale menu until that user's next page load.
func (s *Service) announcePermissionChange(ctx context.Context, roleID string) {
	userIDs, err := s.roleRepository.GetUserIDs(ctx, roleID)
	if err != nil {
		config.AppLogger.WithError(err).Error("failed to list users for permission change broadcast")
		return
	}
	s.broadcaster.BroadcastPermissionsChanged(userIDs)
}

func (s *Service) GetAllPermissions(ctx context.Context) ([]Permission, error) {
	return s.permissionRepository.GetAll(ctx)
}

func (s *Service) GetAllRoles(ctx context.Context) ([]Role, error) {
	return s.roleRepository.GetAll(ctx)
}

func (s *Service) GetRoleById(ctx context.Context, id string) (*Role, error) {
	return s.roleRepository.GetById(ctx, id)
}

func (s *Service) AddNewRole(ctx context.Context, request *StoreRoleRequest) (*Role, error) {
	permissions, err := s.resolvePermissions(ctx, request.PermissionIDs)
	if err != nil {
		return nil, err
	}

	role := Role{Name: request.Name, IsMutable: true, Permissions: permissions}
	if err := s.roleRepository.AddNewData(ctx, &role); err != nil {
		return nil, err
	}

	// After the write, not from a GORM hook on Role: hooks run inside the
	// transaction, so a read arriving before the COMMIT would cache the rows as
	// they were and hold them for a whole TTL.
	utils.Forget(utils.ReadCachePrefix)

	goaudit.Init(ctx, "CREATE_ROLE", "create role via rbac.Service.AddNewRole").
		SetEntryObject("roles", role.ID.String()).
		AddSingleTrail("roles", role.ID.String(), nil, map[string]any{"name": role.Name, "is_mutable": role.IsMutable}, nil, nil).
		AddRelationalTrail("role_has_permissions", nil, permissionNames(role.Permissions), nil, nil).
		Execute(ctx)

	return &role, nil
}

func (s *Service) UpdateRoleById(ctx context.Context, id string, request *UpdateRoleRequest) (*Role, error) {
	var role *Role
	var permissions []Permission
	var firstError error

	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(2)
	setErr := func(e error) {
		mu.Lock()
		defer mu.Unlock()
		if firstError == nil {
			firstError = e
		}
	}

	go func() {
		defer wg.Done()
		r, err := s.roleRepository.GetById(ctx, id)
		if err != nil {
			setErr(err)
			return
		}

		role = r
	}()

	go func() {
		defer wg.Done()
		p, err := s.resolvePermissions(ctx, request.PermissionIDs)
		if err != nil {
			setErr(err)
		}
		permissions = p
	}()

	wg.Wait()
	if firstError != nil {
		return nil, firstError
	}

	// Captured before either mutation: Name is overwritten below, and
	// UpdateById's own Preload("Permissions") reload at the end overwrites
	// role.Permissions in place with the new set.
	before := map[string]any{"name": role.Name}
	beforePermissions := permissionNames(role.Permissions)

	if role.IsMutable {
		role.Name = request.Name
	}

	if err := s.roleRepository.UpdateById(ctx, role, permissions); err != nil {
		return nil, err
	}

	// The role list carries each role's permissions, and UpdateById replaces
	// those after saving the role itself — so this has to run once the whole
	// transaction is done, not from a hook on the Save inside it.
	utils.Forget(utils.ReadCachePrefix)

	s.announcePermissionChange(ctx, role.ID.String())

	goaudit.Init(ctx, "UPDATE_ROLE", "update role via rbac.Service.UpdateRoleById").
		SetEntryObject("roles", role.ID.String()).
		AddSingleTrail("roles", role.ID.String(), before, map[string]any{"name": role.Name}, nil, nil).
		AddRelationalTrail("role_has_permissions", beforePermissions, permissionNames(role.Permissions), nil, nil).
		Execute(ctx)

	return role, nil
}

func (s *Service) DeleteRoleById(ctx context.Context, id string) error {
	role, err := s.roleRepository.GetById(ctx, id)
	if err != nil {
		return err
	}

	if !role.IsMutable {
		return errors.NonDeleteableException()
	}

	// Collected before the delete: the pivot rows naming these users are gone
	// once it runs.
	userIDs, listErr := s.roleRepository.GetUserIDs(ctx, role.ID.String())
	if listErr != nil {
		config.AppLogger.WithError(listErr).Error("failed to list users for permission change broadcast")
	}

	before := map[string]any{"name": role.Name, "is_mutable": role.IsMutable}
	beforePermissions := permissionNames(role.Permissions)

	if err := s.roleRepository.DeleteById(ctx, role); err != nil {
		return err
	}

	utils.Forget(utils.ReadCachePrefix)

	goaudit.Init(ctx, "DELETE_ROLE", "delete role via rbac.Service.DeleteRoleById").
		SetEntryObject("roles", role.ID.String()).
		AddSingleTrail("roles", role.ID.String(), before, nil, nil, nil).
		AddRelationalTrail("role_has_permissions", beforePermissions, nil, nil, nil).
		Execute(ctx)

	s.broadcaster.BroadcastPermissionsChanged(userIDs)
	return nil
}

func (s *Service) resolvePermissions(ctx context.Context, ids []string) ([]Permission, error) {
	if len(ids) == 0 {
		return []Permission{}, nil
	}

	uuids := make([]uuid.UUID, 0, len(ids))
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, errors.InvalidParsingException("Invalid permission_id: " + raw).WithCause(err)
		}
		uuids = append(uuids, id)
	}

	return s.permissionRepository.GetByIDs(ctx, uuids)
}
func NewService(permissionRepository *PermissionRepository, roleRepository *RoleRepository, broadcaster PermissionBroadcaster) *Service {
	return &Service{permissionRepository, roleRepository, broadcaster}
}
