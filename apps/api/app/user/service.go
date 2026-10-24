package user

import (
	"context"
	"api-kms/errors"
	"api-kms/utils"

	goaudit "github.com/iqbalatma/go-audit"
	"gorm.io/gorm"
)

type Service struct {
	repository *Repository
	// An interface rather than *notification.Hub: that package imports this
	// one, so naming the concrete type here would cycle.
	broadcaster PermissionBroadcaster
}

// PermissionBroadcaster is satisfied by *notification.Hub.
type PermissionBroadcaster interface {
	BroadcastPermissionsChanged(userIDs []string)
}

func NewService(repository *Repository, broadcaster PermissionBroadcaster) *Service {
	return &Service{
		repository:  repository,
		broadcaster: broadcaster,
	}
}

func (s *Service) GetAll(ctx context.Context) ([]User, error) {
	return s.repository.GetAll(ctx)
}

// The Forget call in UpdateById is deliberately not a GORM AfterSave hook on
// User, the way other tables do it — see "Invalidation runs after the write"
// in CLAUDE.md for why hooks are the wrong place for this.
func (s *Service) UpdateById(ctx context.Context, id string, request *UpdateRequest) (*User, error) {
	var user *User
	user, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	// Captured before either mutation below — MarketingId is overwritten in
	// place a few lines down, and the role set only exists in the user_roles
	// pivot (GetById doesn't preload it), so the "before" set has to be read
	// explicitly now rather than off the loaded struct.
	before := map[string]any{"marketing_id": user.MarketingId}
	var beforeRoleIDs []string
	if err := s.repository.db.WithContext(ctx).Table("user_roles").Where("user_id = ?", id).Pluck("role_id", &beforeRoleIDs).Error; err != nil {
		return nil, errors.InternalServerError("failed to load current user roles").WithCause(err)
	}

	user.MarketingId = request.MarketingId
	err = s.repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updatedUser, err := s.repository.WithTransaction(tx).UpdateById(ctx, id, user)
		if err != nil {
			return err
		}

		err = s.repository.WithTransaction(tx).SyncRoles(ctx, updatedUser, request.RoleIDs)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Their roles just changed, so the permission list their frontend cached
	// is stale. Sent after the transaction commits and outside it: a failed
	// push must not undo a role assignment the admin already saved.
	s.broadcaster.BroadcastPermissionsChanged([]string{id})

	// Role changes decide who appears in the sales dropdown.
	utils.Forget(utils.ReadCachePrefix)

	goaudit.Init(ctx, "UPDATE_USER", "update user via user.Service.UpdateById").
		SetEntryObject("users", id).
		AddSingleTrail("users", id, before, map[string]any{"marketing_id": user.MarketingId}, nil, nil).
		AddRelationalTrail("user_roles", beforeRoleIDs, request.RoleIDs, nil, nil).
		Execute(ctx)

	return user, nil
}

// UpdatePassword changes the caller's own password. The current one is
// re-checked here rather than trusted from the session: a token proves who
// they are, not that whoever is holding the keyboard right now knows the
// password they are about to replace.
//
// Nothing is audited but the fact of the change — go-audit records before/after
// values, and a password hash is not a value that belongs in an audit table.
func (s *Service) UpdatePassword(ctx context.Context, user *User, request *UpdatePasswordRequest) error {
	if user.Password == nil || !utils.CheckHash(*user.Password, request.CurrentPassword) {
		return errors.InvalidCredential("Current password is incorrect")
	}

	hashed, err := utils.MakeHash(request.Password)
	if err != nil {
		return errors.InternalServerError("failed to hash password").WithCause(err)
	}

	if err := s.repository.UpdatePassword(ctx, user, hashed); err != nil {
		return err
	}

	goaudit.Init(ctx, "UPDATE_PASSWORD", "change own password via user.Service.UpdatePassword").
		SetEntryObject("users", user.ID.String()).
		Execute(ctx)

	return nil
}
