package category

import (
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/utils"
	"api-kms/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

// Index godoc
// @Summary      List a space's categories as a tree
// @Description  Nested to at most 3 levels, siblings in their stored order. One flat query, assembled server-side — the client never walks the tree to fetch it. Each node carries `document_count` for its whole subtree, which is what clicking it lists; documents FR-9 hides from this caller are left out of every count.
// @Tags         categories
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId  path      string  true  "Space ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=category.CategoryTreeResponse}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /spaces/{spaceId}/categories [get]
func (h *Handler) Index(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	tree, total, uncategorised, err := h.service.GetTreeWithCounts(c.Request.Context(), c.Param("spaceId"), authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Categories retrieved", CategoryTreeResponse{
		Categories:         NewCategoryTreeResource(tree),
		TotalDocumentCount: total,
		UncategorisedCount: uncategorised,
	})
	return nil
}

// Store godoc
// @Summary      Create a category
// @Description  Appended last among its siblings. Refused with 422 if the parent is already at the third level.
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId  path      string                         true  "Space ID"
// @Param        request  body      category.StoreCategoryRequest  true  "Category"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=category.CategoryResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /spaces/{spaceId}/categories [post]
func (h *Handler) Store(c *gin.Context) error {
	var request StoreCategoryRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	category, err := h.service.AddNew(c.Request.Context(), c.Param("spaceId"), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "Category created", NewCategoryResource(category))
	return nil
}

// Update godoc
// @Summary      Update or move a category
// @Description  A full replace: an absent parent_id makes it a root category. Moving it puts it last among its new siblings.
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                          true  "Category ID"
// @Param        request  body      category.UpdateCategoryRequest  true  "Category"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=category.CategoryResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /categories/{id} [put]
func (h *Handler) Update(c *gin.Context) error {
	var request UpdateCategoryRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	category, err := h.service.UpdateById(c.Request.Context(), c.Param("id"), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Category updated", NewCategoryResource(category))
	return nil
}

// Destroy godoc
// @Summary      Delete a category
// @Description  Its subtree goes with it. Documents filed under it are kept and left uncategorised — a document outlives the shelf it was on.
// @Tags         categories
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Category ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /categories/{id} [delete]
func (h *Handler) Destroy(c *gin.Context) error {
	if err := h.service.DeleteById(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Category deleted", nil)
	return nil
}

// Reorder godoc
// @Summary      Reorder one level of a space's category tree
// @Description  Takes the whole ordered sibling list; positions are renumbered 0..n-1 in one transaction. A partial list is refused with 422.
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId  path      string                             true  "Space ID"
// @Param        request  body      category.ReorderCategoriesRequest  true  "Ordered sibling ids"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]category.CategoryResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /spaces/{spaceId}/categories/order [patch]
func (h *Handler) Reorder(c *gin.Context) error {
	var request ReorderCategoriesRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	spaceID := c.Param("spaceId")
	if err := h.service.Reorder(c.Request.Context(), spaceID, &request); err != nil {
		return err
	}

	// Answers with the whole tree rather than an empty 200: positions shifted
	// across the level, and a client reassembling them would be a second
	// implementation of the ordering rules waiting to disagree.
	authUser := c.MustGet("user").(*user.User)
	tree, total, uncategorised, err := h.service.GetTreeWithCounts(c.Request.Context(), spaceID, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Categories reordered", CategoryTreeResponse{
		Categories:         NewCategoryTreeResource(tree),
		TotalDocumentCount: total,
		UncategorisedCount: uncategorised,
	})
	return nil
}

func NewHandler(service *Service) *Handler {
	return &Handler{service}
}
