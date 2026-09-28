package rbac

type StoreRoleRequest struct {
	Name          string   `form:"name" json:"name" binding:"required"`
	PermissionIDs []string `form:"permission_ids" json:"permission_ids"`
}

type UpdateRoleRequest struct {
	Name          string   `form:"name" json:"name" binding:"required"`
	PermissionIDs []string `form:"permission_ids" json:"permission_ids"`
}
