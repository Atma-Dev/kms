package rbac

import (
	"api-kms/utils"
)

type RoleResource struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	IsMutable   bool                 `json:"is_mutable"`
	Permissions []PermissionResource `json:"permissions"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

func NewRoleResource(role *Role) RoleResource {
	permissions := make([]PermissionResource, len(role.Permissions))
	for i, p := range role.Permissions {
		permissions[i] = NewPermissionResource(p)
	}

	return RoleResource{
		ID:          role.ID.String(),
		Name:        role.Name,
		IsMutable:   role.IsMutable,
		Permissions: permissions,
		CreatedAt:   utils.FormatDateTimeVal(role.CreatedAt),
		UpdatedAt:   utils.FormatDateTimeVal(role.UpdatedAt),
	}
}

func NewRoleResourceCollection(roles []Role) []RoleResource {
	result := make([]RoleResource, len(roles))
	for i, role := range roles {
		result[i] = NewRoleResource(&role)
	}
	return result
}

type PermissionResource struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	FeatureGroup string `json:"feature_group"`
	CreatedAt    string `json:"created_at"`
}

func NewPermissionResource(p Permission) PermissionResource {
	return PermissionResource{
		ID:           p.ID.String(),
		Name:         p.Name,
		Description:  p.Description,
		FeatureGroup: p.FeatureGroup,
		CreatedAt:    utils.FormatDateTimeVal(p.CreatedAt),
	}
}

func NewPermissionResourceCollection(ps []Permission) []PermissionResource {
	result := make([]PermissionResource, len(ps))
	for i, p := range ps {
		result[i] = NewPermissionResource(p)
	}
	return result
}
