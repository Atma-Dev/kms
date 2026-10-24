package search

import (
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

// Search godoc
// @Summary      Search documents
// @Description  Full-text search over title and content (FR-1), filtered and sorted (FR-2), and limited to what this caller may see (FR-3). Each result carries a `snippet` with the matched words wrapped in `<mark>`. Relevance weights a title match above a body match; with no `q` the list sorts by newest.
// @Tags         search
// @Produce      json
// @Security     BearerAuth
// @Param        q            query     string  false  "What to search for"
// @Param        space_id     query     string  false  "Filter by space"
// @Param        category_id  query     string  false  "Filter by category, including everything under it"
// @Param        tag          query     string  false  "Filter by tag ID"
// @Param        author_id    query     string  false  "Filter by author"
// @Param        from         query     string  false  "Updated on or after (2006-01-02)"
// @Param        to           query     string  false  "Updated on or before (2006-01-02)"
// @Param        sort         query     string  false  "Ordering"  Enums(relevance, newest, popular)
// @Param        page         query     int     false  "Page number"
// @Param        per_page     query     int     false  "Rows per page"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]search.ResultResource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /search [get]
func (h *Handler) Search(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	results, meta, err := h.service.Search(c, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Search results retrieved", utils.Payload{
		Data: NewResultResourceCollection(results),
		Meta: meta,
	})
	return nil
}

// Suggest godoc
// @Summary      Autocomplete document titles
// @Description  Titles starting with what has been typed so far (FR-6), most read first. A prefix match, not the full-text index — a dropdown completes the word being typed rather than finding it anywhere.
// @Tags         search
// @Produce      json
// @Security     BearerAuth
// @Param        q         query     string  true   "What has been typed so far"
// @Param        space_id  query     string  false  "Limit to one space"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]search.SuggestionResource}}
// @Failure      401  {object}  utils.HTTPError
// @Router       /search/suggest [get]
func (h *Handler) Suggest(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	documents, err := h.service.Suggest(c, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Suggestions retrieved", NewSuggestionResourceCollection(documents))
	return nil
}

// History godoc
// @Summary      Recent searches
// @Description  This caller's own last queries (FR-5), most recently searched first — what the search box offers when it is focused and empty.
// @Tags         search
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]search.HistoryResource}}
// @Failure      401  {object}  utils.HTTPError
// @Router       /search/history [get]
func (h *Handler) History(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	histories, err := h.service.Histories(c.Request.Context(), authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Search history retrieved", NewHistoryResourceCollection(histories))
	return nil
}

// DestroyHistory godoc
// @Summary      Forget one past search
// @Description  Removes one entry from this caller's own history. Scoped to them — an id from somebody else's history is a 404.
// @Tags         search
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "History entry ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /search/history/{id} [delete]
func (h *Handler) DestroyHistory(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	if err := h.service.DeleteHistory(c.Request.Context(), authUser, c.Param("id")); err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Search history entry deleted", nil)
	return nil
}

func NewHandler(service *Service) *Handler {
	return &Handler{service}
}
