package rbac

import (
	"context"
	stderrors "errors"
	"api-kms/errors"
	"api-kms/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleRepository struct {
	db *gorm.DB
}

// Cached: /api/management/roles and /api/lookups/roles run this same query and
// only differ in what they map it to. Service writes clear it.
func (r *RoleRepository) GetAll(ctx context.Context) ([]Role, error) {
	return utils.Remember(utils.ReadCachePrefix+"roles", utils.SharedTableCacheTTL, func() ([]Role, error) {
		var roles []Role
		if err := r.db.WithContext(ctx).Preload("Permissions").Order("name ASC").Find(&roles).Error; err != nil {
			return nil, errors.InternalServerError("failed to get roles").WithCause(err)
		}
		return roles, nil
	})
}

func (r *RoleRepository) GetById(ctx context.Context, id string) (*Role, error) {
	var role Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("role not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get role by id").WithCause(err)
	}
	return &role, nil
}

func (r *RoleRepository) AddNewData(ctx context.Context, role *Role) error {
	if err := r.db.WithContext(ctx).Create(role).Error; err != nil {
		return errors.InternalServerError("failed to create role").WithCause(err)
	}
	return nil
}

func (r *RoleRepository) UpdateById(ctx context.Context, role *Role, permissions []Permission) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(role).Error; err != nil {
			return errors.InternalServerError("failed to update role").WithCause(err)
		}

		if err := tx.Model(role).Association("Permissions").Replace(permissions); err != nil {
			return errors.InternalServerError("failed to replace role permissions").WithCause(err)
		}

		if err := tx.Preload("Permissions").First(role, role.ID).Error; err != nil {
			return errors.InternalServerError("failed to reload role after update").WithCause(err)
		}

		return nil
	})

	// Errors from the closure are already translated. A raw Begin/Commit failure
	// falls through to ErrorHandler, which renders a generic 500 and logs it.
	return err
}

func (r *RoleRepository) DeleteById(ctx context.Context, role *Role) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(role).Association("Permissions").Unscoped().Clear(); err != nil {
			return errors.InternalServerError("failed to clear role permissions").WithCause(err)
		}
		if err := tx.Delete(role).Error; err != nil {
			return errors.InternalServerError("failed to delete role").WithCause(err)
		}
		return nil
	})

	return err
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

type PermissionRepository struct {
	db *gorm.DB
}

// Cached, and nothing invalidates it: this repository is read only and the rows
// come from database/seeders, which runs as its own binary. The TTL is the
// whole story here.
func (pr *PermissionRepository) GetAll(ctx context.Context) ([]Permission, error) {
	return utils.Remember(utils.ReadCachePrefix+"permissions", utils.SharedTableCacheTTL, func() ([]Permission, error) {
		var permissions []Permission
		if err := pr.db.WithContext(ctx).Find(&permissions).Error; err != nil {
			return nil, errors.InternalServerError("failed to get permissions").WithCause(err)
		}
		return permissions, nil
	})
}

func (pr *PermissionRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]Permission, error) {
	var permissions []Permission
	if len(ids) == 0 {
		return permissions, nil
	}
	if err := pr.db.WithContext(ctx).Where("id IN ?", ids).Find(&permissions).Error; err != nil {
		return nil, errors.InternalServerError("failed to get permissions by ids").WithCause(err)
	}
	return permissions, nil
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db}
}

// GetUserIDs lists the users holding this role, so a permission change can be
// pushed to exactly them instead of broadcast to everyone.
func (r *RoleRepository) GetUserIDs(ctx context.Context, roleID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Where("role_id = ?", roleID).
		Pluck("user_id", &ids).Error
	if err != nil {
		return nil, errors.InternalServerError("failed to get user ids by role").WithCause(err)
	}
	return ids, nil
}
