package rbac

import "api-kms/app"

type Permission struct {
	app.BaseModel `gorm:"embedded"`
	Name          string `gorm:"column:name" json:"name"`
	Description   string `gorm:"column:description" json:"description"`
	FeatureGroup  string `gorm:"column:feature_group" json:"feature_group"`
	Roles         []Role `gorm:"many2many:role_has_permissions;"`
}
