package document

import (
	"api-kms/app/protection"
	"api-kms/app/tag"
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
// @Summary      List documents in a space
// @Description  Pinned documents sort first whatever the chosen order. `search` matches title and excerpt only — full-text search over the content is Modul 3.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId      path      string  true   "Space ID"
// @Param        page         query     int     false  "Page number"
// @Param        per_page     query     int     false  "Rows per page"
// @Param        status       query     string  false  "Filter by status"  Enums(draft, in_review, published, archived)
// @Param        category_id  query     string  false  "Filter by category"
// @Param        author_id    query     string  false  "Filter by author"
// @Param        search       query     string  false  "Match on title or excerpt"
// @Param        sort         query     string  false  "Sort column"  Enums(updated_at, view_count, title, created_at)
// @Param        direction    query     string  false  "Sort direction"  Enums(asc, desc)
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]document.DocumentResource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /spaces/{spaceId}/documents [get]
func (h *Handler) Index(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	documents, meta, err := h.service.GetAllPaginated(c, c.Param("spaceId"), authUser)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Documents retrieved", utils.Payload{
		Data: NewDocumentResourceCollection(documents),
		Meta: meta,
	})
	return nil
}

// Show godoc
// @Summary      Get a document with its content
// @Description  Counts the read at most once per hour per reader, and only for published documents. The response carries `available_actions` — the lifecycle moves this caller may make right now.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.DocumentDetailResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id} [get]
func (h *Handler) Show(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	doc, err := h.service.GetById(c.Request.Context(), c.Param("id"), authUser)
	if err != nil {
		return err
	}

	// Modul 4 FR-4, and the one place a view is recorded. Every open, not one
	// an hour: the throttle on documents.view_count exists to keep a statistic
	// honest, while this is the trail an investigation reads, and a trail that
	// drops 59 minutes of every hour cannot answer "how often did they open
	// it". The two live in different tables for exactly that reason.
	//
	// The page images (FR-3) deliberately do not log a second time — they are
	// how this one view is drawn, not another reach for the document.
	h.service.RecordAccess(c, doc.ID, authUser, protection.ActionView)

	utils.ResponseJSON(c, enums.SUCCESS, "Document retrieved", h.detail(c, doc))
	return nil
}

// Related godoc
// @Summary      Related documents
// @Description  The documents a reader of this one is most likely to want next (FR-4): most shared tags first, same category as the tie-breaker. Published documents in the same space only, filtered by what this caller may see.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]document.DocumentResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/related [get]
func (h *Handler) Related(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	related, err := h.service.Related(c.Request.Context(), c.Param("id"), authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Related documents retrieved", NewDocumentResourceCollection(related))
	return nil
}

// Store godoc
// @Summary      Create a draft
// @Description  Always starts as a draft — publishing is a separate act with its own permission, never a field on the create form.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId  path      string                          true  "Space ID"
// @Param        request  body      document.StoreDocumentRequest   true  "Document"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=document.DocumentDetailResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /spaces/{spaceId}/documents [post]
func (h *Handler) Store(c *gin.Context) error {
	var request StoreDocumentRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	authUser := c.MustGet("user").(*user.User)
	doc, err := h.service.AddNew(c.Request.Context(), c.Param("spaceId"), &request, authUser.ID)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "Document created", h.detail(c, doc))
	return nil
}

// Update godoc
// @Summary      Update a document
// @Description  On a document that has ever been published, the content being replaced is snapshotted as a new version first — in the same transaction as the write.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                          true  "Document ID"
// @Param        request  body      document.UpdateDocumentRequest  true  "Document"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.DocumentDetailResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /documents/{id} [put]
func (h *Handler) Update(c *gin.Context) error {
	var request UpdateDocumentRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	authUser := c.MustGet("user").(*user.User)
	doc, err := h.service.UpdateById(c.Request.Context(), c.Param("id"), &request, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Document updated", h.detail(c, doc))
	return nil
}

// Destroy godoc
// @Summary      Delete a document
// @Description  Soft delete: the row stays so the audit trail keeps pointing at something real.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id} [delete]
func (h *Handler) Destroy(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	if err := h.service.DeleteById(c.Request.Context(), c.Param("id"), authUser); err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Document deleted", nil)
	return nil
}

// Transition godoc
// @Summary      Move a document through its lifecycle
// @Description  One handler per action, bound in routes/route.go: submit-review, approve, reject, publish, unpublish, archive, restore. An illegal move answers 422 naming both statuses. `reason` is required on reject.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                        true   "Document ID"
// @Param        request  body      document.TransitionRequest    false  "Reason, required when rejecting"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.DocumentDetailResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /documents/{id}/{action} [post]
func (h *Handler) Transition(action Action) func(*gin.Context) error {
	return func(c *gin.Context) error {
		// The body is optional on every action but reject, whose own check
		// runs in the service — so a missing body must not be a bind error.
		var request TransitionRequest
		if c.Request.ContentLength > 0 {
			if err := validator.BindAndValidate(c, &request); err != nil {
				return err
			}
		}

		authUser := c.MustGet("user").(*user.User)
		doc, err := h.service.Transition(c.Request.Context(), c.Param("id"), action, &request, authUser)
		if err != nil {
			return err
		}

		utils.ResponseJSON(c, enums.SUCCESS, "Document "+string(action), h.detail(c, doc))
		return nil
	}
}

// Versions godoc
// @Summary      List a document's versions
// @Description  Newest first, without content — the list is a sidebar of numbers and notes.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]document.VersionResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/versions [get]
func (h *Handler) Versions(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	versions, err := h.service.GetVersions(c.Request.Context(), c.Param("id"), authUser)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Document versions retrieved", NewVersionResourceCollection(versions))
	return nil
}

// Version godoc
// @Summary      Get one version with its content
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id             path      string  true  "Document ID"
// @Param        versionNumber  path      int     true  "Version number, as shown in the list"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.VersionDetailResource}}
// @Failure      400  {object}  utils.HTTPError
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/versions/{versionNumber} [get]
func (h *Handler) Version(c *gin.Context) error {
	number, err := parseVersionNumber(c.Param("versionNumber"))
	if err != nil {
		return err
	}

	authUser := c.MustGet("user").(*user.User)

	version, err := h.service.GetVersion(c.Request.Context(), c.Param("id"), number, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Document version retrieved", NewVersionDetailResource(version))
	return nil
}

// Compare godoc
// @Summary      Diff two versions of a document
// @Description  Line-by-line over the visible text, not the raw JSON. Either side may be `current` instead of a version number — snapshots are taken *before* an overwrite, so the live content is not itself a version row.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string  true  "Document ID"
// @Param        from  query     string  true  "Version number, or `current`"
// @Param        to    query     string  true  "Version number, or `current`"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]document.DiffLine}}
// @Failure      400  {object}  utils.HTTPError
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/versions/compare [get]
func (h *Handler) Compare(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	diff, err := h.service.Compare(c.Request.Context(), c.Param("id"), c.Query("from"), c.Query("to"), authUser)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Versions compared", diff)
	return nil
}

// Rollback godoc
// @Summary      Roll a document back to an earlier version
// @Description  Snapshots the content it replaces first, so a rollback is itself undoable.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id             path      string                      true   "Document ID"
// @Param        versionNumber  path      int                         true   "Version number to restore"
// @Param        request        body      document.RollbackRequest    false  "Change note"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.DocumentDetailResource}}
// @Failure      400  {object}  utils.HTTPError
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/versions/{versionNumber}/rollback [post]
func (h *Handler) Rollback(c *gin.Context) error {
	number, err := parseVersionNumber(c.Param("versionNumber"))
	if err != nil {
		return err
	}

	var request RollbackRequest
	if c.Request.ContentLength > 0 {
		if err := validator.BindAndValidate(c, &request); err != nil {
			return err
		}
	}

	authUser := c.MustGet("user").(*user.User)
	doc, err := h.service.Rollback(c.Request.Context(), c.Param("id"), number, &request, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Document rolled back", h.detail(c, doc))
	return nil
}

// detail assembles the single-document response: the document, the tags on it,
// and whether it is restricted. One place does it so a field added later
// cannot reach four of the five endpoints that return a document.
//
// A failure loading tags is not fatal — the document is what was asked for,
// and answering 500 because its chips would not load helps nobody.
func (h *Handler) detail(c *gin.Context, doc *Document) DocumentDetailResource {
	authUser := c.MustGet("user").(*user.User)
	ctx := c.Request.Context()

	tags, err := h.service.TagsFor(ctx, doc.ID.String())
	if err != nil {
		tags = nil
	}
	grants, err := h.service.repository.GrantsFor(ctx, doc.ID.String())
	if err != nil {
		grants = nil
	}

	return NewDocumentDetailResource(doc, authUser, tags, len(grants) > 0, h.service.ContentVisibleTo(ctx, doc, authUser))
}

func NewHandler(service *Service) *Handler {
	return &Handler{service}
}

// SyncTags godoc
// @Summary      Replace a document's tags
// @Description  Takes the whole set by name; a name nobody has used becomes a tag (FR-6). Answers to the document's edit rule, not to a tag permission — retagging is an edit of the document.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                       true  "Document ID"
// @Param        request  body      tag.SyncDocumentTagsRequest  true  "Tag names"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]tag.TagResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/tags [post]
func (h *Handler) SyncTags(c *gin.Context) error {
	var request tag.SyncDocumentTagsRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	authUser := c.MustGet("user").(*user.User)
	tags, err := h.service.SyncTags(c.Request.Context(), c.Param("id"), &request, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Document tags updated", tag.NewTagResourceCollection(tags))
	return nil
}

// Access godoc
// @Summary      List a document's access grants
// @Description  Empty means the document is unrestricted and ordinary permissions apply. Any rows mean it is restricted to the subjects listed (FR-9).
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]document.AccessResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/access [get]
func (h *Handler) Access(c *gin.Context) error {
	grants, err := h.service.Grants(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Document access retrieved", NewAccessResourceCollection(grants))
	return nil
}

// StoreAccess godoc
// @Summary      Restrict a document to a user or role
// @Description  The first grant narrows the document to whoever is named; it never widens one. Holders of documents.access.manage are never locked out.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                          true  "Document ID"
// @Param        request  body      document.StoreAccessRequest     true  "Grant"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=document.AccessResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      409  {object}  utils.HTTPError
// @Router       /documents/{id}/access [post]
func (h *Handler) StoreAccess(c *gin.Context) error {
	var request StoreAccessRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	grant, err := h.service.AddGrant(c.Request.Context(), c.Param("id"), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "Document access granted", NewAccessResource(grant))
	return nil
}

// DestroyAccess godoc
// @Summary      Revoke one access grant
// @Description  Removing the last grant makes the document unrestricted again.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id        path      string  true  "Document ID"
// @Param        accessId  path      string  true  "Access grant ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/access/{accessId} [delete]
func (h *Handler) DestroyAccess(c *gin.Context) error {
	if err := h.service.RevokeGrant(c.Request.Context(), c.Param("id"), c.Param("accessId")); err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Document access revoked", nil)
	return nil
}

// Pin godoc
// @Summary      Pin or unpin a document
// @Description  Pinned documents sort first in their space whatever order is chosen. Answers to the document's edit rule, and carries no content — flipping the flag cannot overwrite someone else's edit.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                       true  "Document ID"
// @Param        request  body      document.PinDocumentRequest  true  "Pin state"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.DocumentDetailResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/pin [patch]
func (h *Handler) Pin(c *gin.Context) error {
	var request PinDocumentRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	authUser := c.MustGet("user").(*user.User)
	doc, err := h.service.SetPinned(c.Request.Context(), c.Param("id"), *request.IsPinned, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Document updated", h.detail(c, doc))
	return nil
}

// Deleted godoc
// @Summary      List a space's deleted documents
// @Description  The bin. Gated on documents.destroy, which is strictly more than reading — and restricted documents are not filtered out, or one could never be restored by anyone.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        spaceId   path      string  true   "Space ID"
// @Param        page      query     int     false  "Page number"
// @Param        per_page  query     int     false  "Rows per page"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]document.DocumentResource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /spaces/{spaceId}/documents/deleted [get]
func (h *Handler) Deleted(c *gin.Context) error {
	documents, meta, err := h.service.GetDeletedPaginated(c, c.Param("spaceId"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Deleted documents retrieved", utils.Payload{
		Data: NewDocumentResourceCollection(documents),
		Meta: meta,
	})
	return nil
}

// Undelete godoc
// @Summary      Restore a deleted document
// @Description  Comes back as a draft even if it was published when deleted: putting content back in front of readers without anyone looking at why it was deleted is the one outcome nobody asked for.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.DocumentDetailResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/undelete [post]
func (h *Handler) Undelete(c *gin.Context) error {
	doc, err := h.service.Undelete(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Document restored", h.detail(c, doc))
	return nil
}
