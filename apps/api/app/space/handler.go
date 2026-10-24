package space

import (
	"api-kms/app/protection"
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/errors"
	"api-kms/utils"
	"api-kms/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

// Index godoc
// @Summary      List spaces
// @Description  Archived spaces are left out unless `include_archived=1`.
// @Tags         spaces
// @Produce      json
// @Security     BearerAuth
// @Param        page              query     int     false  "Page number"
// @Param        per_page          query     int     false  "Rows per page"
// @Param        search            query     string  false  "Match on name"
// @Param        include_archived  query     string  false  "1 to include archived spaces"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]space.SpaceResource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /spaces [get]
func (h *Handler) Index(c *gin.Context) error {
	spaces, meta, err := h.service.GetAllPaginated(c)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Spaces retrieved", utils.Payload{
		Data: NewSpaceResourceCollection(spaces),
		Meta: meta,
	})
	return nil
}

// Show godoc
// @Summary      Get a space
// @Tags         spaces
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId  path      string  true  "Space ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=space.SpaceResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /spaces/{spaceId} [get]
func (h *Handler) Show(c *gin.Context) error {
	space, err := h.service.GetById(c.Request.Context(), c.Param("spaceId"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Space retrieved", NewSpaceResource(space))
	return nil
}

// Store godoc
// @Summary      Create a space
// @Description  The slug is derived from the name, made unique with a -2, -3 suffix on collision.
// @Tags         spaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      space.StoreSpaceRequest  true  "Space"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=space.SpaceResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /spaces [post]
func (h *Handler) Store(c *gin.Context) error {
	var request StoreSpaceRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	authUser := c.MustGet("user").(*user.User)
	space, err := h.service.AddNew(c.Request.Context(), &request, authUser.ID)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "Space created", NewSpaceResource(space))
	return nil
}

// Update godoc
// @Summary      Update a space
// @Description  The slug only moves when the name does — links to a space that is merely being re-described keep working.
// @Tags         spaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId  path      string                    true  "Space ID"
// @Param        request  body      space.UpdateSpaceRequest  true  "Space"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=space.SpaceResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /spaces/{spaceId} [put]
func (h *Handler) Update(c *gin.Context) error {
	var request UpdateSpaceRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	space, err := h.service.UpdateById(c.Request.Context(), c.Param("spaceId"), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Space updated", NewSpaceResource(space))
	return nil
}

// Destroy godoc
// @Summary      Archive a space
// @Description  Archives rather than deletes: a space owns documents, and dropping the row would take them with it.
// @Tags         spaces
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId  path      string  true  "Space ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=space.SpaceResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /spaces/{spaceId} [delete]
func (h *Handler) Destroy(c *gin.Context) error {
	space, err := h.service.Archive(c.Request.Context(), c.Param("spaceId"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Space archived", NewSpaceResource(space))
	return nil
}

func NewHandler(service *Service) *Handler {
	return &Handler{service}
}

// ProtectionSetting godoc
// @Summary      Set a space's default protection level
// @Description  FR-6: the level every document in this space is served at unless it carries an override. There is no matching GET — the space detail response already carries `protection_level`, and a second endpoint returning one field of a resource the client is already holding is one more thing to keep in step.
// @Tags         spaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId  path      string                                true  "Space ID"
// @Param        request  body      protection.SetProtectionLevelRequest  true  "Level"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=space.SpaceResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /spaces/{spaceId}/protection-setting [put]
func (h *Handler) ProtectionSetting(c *gin.Context) error {
	var request protection.SetProtectionLevelRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}
	// A space is what every document without an override inherits from, so it
	// has to hold a value. Null here would leave documents inheriting nothing.
	if request.ProtectionLevel == nil {
		return errors.InvalidStatusException("a space must have a protection level").
			WithErrors(map[string][]string{"protection_level": {"is required"}})
	}

	space, err := h.service.SetProtectionLevel(c.Request.Context(), c.Param("spaceId"), *request.ProtectionLevel)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Protection level updated", NewSpaceResource(space))
	return nil
}
