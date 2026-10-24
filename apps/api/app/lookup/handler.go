package lookup

import (
	"api-kms/enums"
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

// Get godoc
// @Summary      Get select options for one resource
// @Description  Backs a filter or form dropdown: id and name only. Each resource is gated by its own lookups.* permission, deliberately separate from the *.show permission of the underlying resource — seeing names in a dropdown is a smaller capability than managing the records behind them.
// @Tags         lookups
// @Produce      json
// @Security     BearerAuth
// @Param        space_id  query     string  false  "Space ID — required for the categories lookup"
// @Param        resource  path      string  true  "Lookup resource"  Enums(users, roles, permissions, spaces, categories, tags)
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]lookup.Option}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /lookups/{resource} [get]
func (h *Handler) Get(resource string) func(*gin.Context) error {
	return func(c *gin.Context) error {
		// scope is the categories lookup's space_id; every other resource ignores it.
		options, err := h.service.Get(c.Request.Context(), resource, c.Query("space_id"))
		if err != nil {
			return err
		}
		utils.ResponseJSON(c, enums.SUCCESS, "Lookup retrieved", options)
		return nil
	}
}

func NewHandler(service *Service) *Handler {
	return &Handler{service}
}
