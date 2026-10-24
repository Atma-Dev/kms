package user

import (
	"api-kms/enums"
	"api-kms/utils"
	"api-kms/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Index godoc
// @Summary      List users
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Description  The whole list, unpaginated — one row per employee, filtered in the browser.
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]user.Resource}}
// @Failure      401  {object}  utils.HTTPError
// @Router       /management/users [get]
func (h *Handler) Index(c *gin.Context) error {
	users, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Users retrieved", NewResourceCollection(users))
	return nil
}

// Update godoc
// @Summary      Update a user's roles/marketing assignment
// @Tags         management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string              true  "User ID"
// @Param        request  body      user.UpdateRequest  true  "Fields to update"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=user.Resource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/users/{id} [patch]
func (h *Handler) Update(c *gin.Context) error {
	var request UpdateRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	user, err := h.service.UpdateById(c.Request.Context(), c.Param("id"), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Update user by id successfully", NewResource(user))

	return nil
}
