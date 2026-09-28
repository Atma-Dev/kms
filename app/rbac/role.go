package rbac

import "api-kms/app"

type Role struct {
	app.BaseModel `gorm:"embedded"`
	Name          string       `gorm:"column:name" json:"name"`
	IsMutable     bool         `gorm:"column:is_mutable" json:"is_mutable"`
	Permissions   []Permission `gorm:"many2many:role_has_permissions;"`
}
