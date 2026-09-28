package rbac

import (
	"api-kms/enums"
	"api-kms/utils"
	"api-kms/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

// PermissionIndex godoc
// @Summary      List permissions
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]rbac.PermissionResource}}
// @Failure      401  {object}  utils.HTTPError
// @Router       /management/permissions [get]
func (h *Handler) PermissionIndex(c *gin.Context) error {
	permissions, err := h.service.GetAllPermissions(c.Request.Context())
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Permissions retrieved", NewPermissionResourceCollection(permissions))
	return nil
}

// RoleIndex godoc
// @Summary      List roles
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]rbac.RoleResource}}
// @Failure      401  {object}  utils.HTTPError
// @Router       /management/roles [get]
func (h *Handler) RoleIndex(c *gin.Context) error {
	roles, err := h.service.GetAllRoles(c.Request.Context())
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Roles retrieved", NewRoleResourceCollection(roles))
	return nil
}

// RoleShow godoc
// @Summary      Get a role
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Role ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=rbac.RoleResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /management/roles/{id} [get]
func (h *Handler) RoleShow(c *gin.Context) error {
	role, err := h.service.GetRoleById(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Role retrieved", NewRoleResource(role))
	return nil
}

// RoleStore godoc
// @Summary      Create a role
// @Tags         management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      rbac.StoreRoleRequest  true  "Role"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=rbac.RoleResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/roles [post]
func (h *Handler) RoleStore(c *gin.Context) error {
	var request StoreRoleRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	role, err := h.service.AddNewRole(c.Request.Context(), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "Role created", NewRoleResource(role))
	return nil
}

// RoleUpdate godoc
// @Summary      Update a role
// @Tags         management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                  true  "Role ID"
// @Param        request  body      rbac.UpdateRoleRequest  true  "Role"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=rbac.RoleResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/roles/{id} [patch]
func (h *Handler) RoleUpdate(c *gin.Context) error {
	var request UpdateRoleRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}
	role, err := h.service.UpdateRoleById(c.Request.Context(), c.Param("id"), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Role updated", NewRoleResource(role))
	return nil
}

// RoleDestroy godoc
// @Summary      Delete a role
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Role ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /management/roles/{id} [delete]
func (h *Handler) RoleDestroy(c *gin.Context) error {
	if err := h.service.DeleteRoleById(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Role deleted", nil)
	return nil
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service,
	}
}
