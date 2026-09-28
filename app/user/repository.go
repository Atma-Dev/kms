package user

import (
	"context"
	stderrors "errors"
	"api-kms/app/rbac"
	"api-kms/enums"
	"api-kms/errors"
	"api-kms/utils"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithTransaction(tx *gorm.DB) *Repository {
	return &Repository{db: tx}
}

// GetAll returns every user, roles included. The list is small — one row per
// employee — so it is not paginated: /account/users fetches it once and filters
// in the browser, the same way the roles and master lists work. That also makes
// it cacheable, which a per-page, per-search query never was.
func (r *Repository) GetAll(ctx context.Context) ([]User, error) {
	return utils.Remember(utils.ReadCachePrefix+"users", utils.SharedTableCacheTTL, func() ([]User, error) {
		var users []User
		if err := r.db.WithContext(ctx).
			Preload("Roles").
			Order("first_name ASC, id ASC").
			Find(&users).Error; err != nil {
			return nil, errors.InternalServerError("failed to get users").WithCause(err)
		}
		return users, nil
	})
}

func (r *Repository) UpdateById(ctx context.Context, id string, user *User) (*User, error) {
	err := r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", user.ID).
		Updates(user).Error
	if err != nil {
		return nil, errors.InternalServerError("failed to update user").WithCause(err)
	}

	return user, nil
}

// UpdatePassword writes the hash with a bare column update rather than
// Updates(user): a struct update would also flush whatever else is on the
// in-memory user, and this is called with the session's own copy.
func (r *Repository) UpdatePassword(ctx context.Context, user *User, hashed string) error {
	if err := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", user.ID).Update("password", hashed).Error; err != nil {
		return errors.InternalServerError("failed to update password").WithCause(err)
	}
	user.Password = &hashed
	return nil
}

func (r *Repository) SyncRoles(ctx context.Context, user *User, roleIds []string) error {
	var roles []rbac.Role
	err := r.db.WithContext(ctx).Where("id IN ?", roleIds).Find(&roles).Error
	if err != nil {
		return errors.InternalServerError("failed to get roles for sync").WithCause(err)
	}

	err = r.db.WithContext(ctx).Model(user).Association("Roles").Replace(roles)
	if err != nil {
		return errors.InternalServerError("failed to replace user roles").WithCause(err)
	}

	return nil
}
func (r *Repository) GetById(ctx context.Context, id string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.EmptyDataException("user not found").WithCause(err)
		}
		return nil, errors.InternalServerError("failed to get user by id").WithCause(err)
	}

	return &user, nil
}

// GetByRoleLookup lists the users holding one role, as select options. Takes
// the role by name so callers name the role they mean (enums.GURU) instead of
// carrying a seeded uuid around.
//
// Cached per role.
func (r *Repository) GetByRoleLookup(ctx context.Context, role enums.Role) ([]User, error) {
	return utils.Remember(utils.ReadCachePrefix+"users:by-role:"+string(role), utils.SharedTableCacheTTL, func() ([]User, error) {
		var users []User
		err := r.db.WithContext(ctx).
			Select("users.id", "users.first_name", "users.last_name", "users.email", "users.number").
			Joins("INNER JOIN user_roles ON user_roles.user_id = users.id").
			Joins("INNER JOIN roles ON roles.id = user_roles.role_id").
			Where("roles.name = ?", string(role)).
			Order("users.first_name ASC, users.id ASC").
			Find(&users).Error
		if err != nil {
			return nil, errors.InternalServerError("failed to get users by role").WithCause(err)
		}
		return users, nil
	})
}

// GetLookup lists users as select options — id and name only, no relations.
// Backs GET /api/lookups/users, which is gated by lookups.users.show rather
// than management.users.show: a filter dropdown needs the names, not the right
// to manage the people behind them.
func (r *Repository) GetLookup(ctx context.Context) ([]User, error) {
	return utils.Remember(utils.ReadCachePrefix+"users:lookup", utils.SharedTableCacheTTL, func() ([]User, error) {
		var users []User
		err := r.db.WithContext(ctx).
			Select("id", "first_name", "last_name", "email").
			Order("first_name ASC, id ASC").
			Find(&users).Error
		if err != nil {
			return nil, errors.InternalServerError("failed to get user lookup").WithCause(err)
		}
		return users, nil
	})
}
