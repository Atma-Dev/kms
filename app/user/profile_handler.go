package user

import (
	"api-kms/enums"
	"api-kms/utils"
	"api-kms/validator"

	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	service *Service
}

func NewProfileHandler(service *Service) *ProfileHandler {
	return &ProfileHandler{service: service}
}

// Me godoc
// @Summary      Get the authenticated user
// @Tags         profile
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=user.Resource}}
// @Failure      401  {object}  utils.HTTPResponse
// @Router       /profiles/me [get]
func (h *ProfileHandler) Me(c *gin.Context) error {
	val, exists := c.Get("user")
	if !exists {
		utils.ResponseJSON(c, enums.ERR_AUTHENTICATION, "Unauthenticated", nil)
		return nil
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Authenticated user", NewResource(val.(*User)))
	return nil
}

// UpdatePassword godoc
// @Summary      Change the authenticated user's own password
// @Description  Requires the current password alongside the new one — a session token proves identity, not that whoever is at the keyboard knows the password being replaced. No permission gates it: it only ever acts on the caller's own row.
// @Tags         profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      user.UpdatePasswordRequest  true  "Current and new password"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      400  {object}  utils.HTTPError
// @Router       /profiles/password [patch]
func (h *ProfileHandler) UpdatePassword(c *gin.Context) error {
	val, exists := c.Get("user")
	if !exists {
		utils.ResponseJSON(c, enums.ERR_AUTHENTICATION, "Unauthenticated", nil)
		return nil
	}

	var request UpdatePasswordRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	if err := h.service.UpdatePassword(c.Request.Context(), val.(*User), &request); err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Password updated", nil)
	return nil
}
