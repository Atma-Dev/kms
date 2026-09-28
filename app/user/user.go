package user

import (
	"api-kms/app"
	"api-kms/app/rbac"
	"api-kms/enums"
	"time"
)

type User struct {
	app.BaseModel `gorm:"embedded"`
	MarketingId   *string `json:"marketing_id" gorm:"column:marketing_id"`
	SsoID         *string `json:"sso_id" gorm:"column:sso_id;unique"`
	// Whether the account is allowed to use this app at all — login refuses
	// anything but Active.
	MembershipStatus MembershipStatus `json:"membership_status" gorm:"column:membership_status;default:ACTIVE"`
	DeactivatedAt    *time.Time       `json:"deactivated_at" gorm:"column:deactivated_at"`
	SyncedAt         *time.Time       `json:"synced_at" gorm:"column:synced_at"`
	Number           *uint64          `json:"number" gorm:"column:number;unique"`
	FirstName        string           `json:"first_name" gorm:"column:first_name"`
	LastName         *string          `json:"last_name" gorm:"column:last_name"`
	Email            string           `json:"email" gorm:"column:email;unique"`
	Password         *string          `json:"-" gorm:"column:password"`
	PhoneNumber      *string          `json:"phone_number" gorm:"column:phone_number;unique"`
	Roles            []rbac.Role      `gorm:"many2many:user_roles"`
}

func (user *User) GetSubjectKey() string {
	return user.ID.String()
}

// HasPermission reads the roles AuthMiddleware preloaded onto the context
// user, so neither the route gate nor a service check needs a query of its own.
//
// It lives on the model rather than in middleware because ownership rules
// (documents.update vs documents.update.all) are decided in the service, where
// the row being edited is known — a route gate cannot see whose document it is.
func (user *User) HasPermission(permission enums.Permission) bool {
	for _, role := range user.Roles {
		if role.Name == string(enums.SUPER_ADMIN) {
			return true
		}
		for _, p := range role.Permissions {
			if p.Name == string(permission) {
				return true
			}
		}
	}
	return false
}

// CanEditDocument answers the PRD's Contributor/Editor split: a contributor
// edits what they authored, an editor edits anyone's. Holding update.all
// implies the narrower grant, so a role only ever needs the one that matches
// what it is.
func (user *User) CanEditDocument(authorID string) bool {
	if user.HasPermission(enums.PermissionDocumentsUpdateAll) {
		return true
	}
	return user.HasPermission(enums.PermissionDocumentsUpdate) && user.ID.String() == authorID
}
