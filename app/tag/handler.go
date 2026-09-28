package tag

import (
	"api-kms/enums"
	"api-kms/utils"
	"api-kms/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

// Index godoc
// @Summary      List tags
// @Description  The whole vocabulary with how many documents each tag labels, so deleting one shows what it would take with it.
// @Tags         tags
// @Produce      json
// @Security     BearerAuth
// @Param        search  query     string  false  "Match on name"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]tag.TagResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /tags [get]
func (h *Handler) Index(c *gin.Context) error {
	tags, counts, err := h.service.GetAllWithCounts(c.Request.Context(), c.Query("search"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Tags retrieved", NewTagResourceCollectionWithCounts(tags, counts))
	return nil
}

// Store godoc
// @Summary      Create a tag
// @Description  Answers with the existing tag when the name is already taken — the caller wanted it to exist, and it does.
// @Tags         tags
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      tag.StoreTagRequest  true  "Tag"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=tag.TagResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /tags [post]
func (h *Handler) Store(c *gin.Context) error {
	var request StoreTagRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	t, err := h.service.AddNew(c.Request.Context(), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "Tag created", NewTagResource(t))
	return nil
}

// Destroy godoc
// @Summary      Delete a tag
// @Description  Removes the label from every document that carried it.
// @Tags         tags
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Tag ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /tags/{id} [delete]
func (h *Handler) Destroy(c *gin.Context) error {
	if err := h.service.DeleteById(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Tag deleted", nil)
	return nil
}

// The document's own tag route lives in app/document.Handler.SyncTags, not
// here: replacing a document's tags is an edit of that document, and only
// app/document knows who authored the row the edit rule turns on.

func NewHandler(service *Service) *Handler {
	return &Handler{service}
}

// Update godoc
// @Summary      Rename a tag
// @Description  Relabels every document already carrying it. A rename onto a name that already exists is refused rather than merged — merging folds one vocabulary entry into another across every document using it, with no undo.
// @Tags         tags
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                true  "Tag ID"
// @Param        request  body      tag.UpdateTagRequest  true  "New name"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=tag.TagResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      409  {object}  utils.HTTPError
// @Router       /tags/{id} [put]
func (h *Handler) Update(c *gin.Context) error {
	var request UpdateTagRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	t, err := h.service.UpdateById(c.Request.Context(), c.Param("id"), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Tag renamed", NewTagResource(t))
	return nil
}
