package document

import (
	"api-kms/app/protection"
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/utils"
	"api-kms/validator"
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// ExportPDF godoc
// @Summary      Export a document as a watermarked PDF
// @Description  The watermark carries the caller's own email and the moment of export, injected at generation (FR-2) — two people exporting the same document get files that differ. A strict document exports as page images rather than text, so nothing can be copied out of the PDF either.
// @Tags         documents
// @Produce      application/pdf
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {file}    binary
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/export-pdf [get]
func (h *Handler) ExportPDF(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	doc, data, err := h.service.ExportPDF(c, c.Param("id"), authUser)
	if err != nil {
		return err
	}

	// The slug, not the title: a title can hold a quote, a newline or a
	// separator, and all three reach a response header from here.
	c.Header("Content-Disposition", `attachment; filename="`+url.PathEscape(doc.Slug)+`.pdf"`)
	// Every copy is watermarked for one reader, so no cache may keep one and
	// hand it to the next.
	c.Header("Cache-Control", "private, no-store")
	c.Data(200, "application/pdf", data)
	return nil
}

// Pages godoc
// @Summary      Render a document as watermarked page images
// @Description  The strict viewer (FR-3): every page is drawn on the server with the watermark already in the pixels, so there is no text to select and no overlay to delete. Returned as data URLs in one response — a browser-issued <img> carries no Authorization header, and this way none is needed.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.DocumentPagesResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/pages [get]
func (h *Handler) Pages(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	_, pages, err := h.service.PageImages(c, c.Param("id"), authUser)
	if err != nil {
		return err
	}

	c.Header("Cache-Control", "private, no-store")
	utils.ResponseJSON(c, enums.SUCCESS, "Document pages rendered", NewDocumentPagesResource(pages))
	return nil
}

// DocumentPagesResource is the strict viewer's payload: one data URL per page,
// in reading order.
//
// Data URLs rather than one request per page. A browser-issued <img src> sends
// no Authorization header, which is the whole reason app/attachment had to
// grow a `?token=` escape hatch; embedding the bytes sidesteps it entirely,
// and the pages were rendered in one pass anyway.
//
// ponytail: base64 costs a third more bytes than the raw JPEGs, all of it in
// one response — about 1.3MB for a five-page document. Fine for a knowledge
// document; if these ever run to fifty pages, move to one authenticated
// request per page rather than trying to shrink the encoding.
type DocumentPagesResource struct {
	Count int      `json:"count"`
	Pages []string `json:"pages"`
}

func NewDocumentPagesResource(pages [][]byte) DocumentPagesResource {
	encoded := make([]string, len(pages))
	for i, page := range pages {
		encoded[i] = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(page)
	}
	return DocumentPagesResource{Count: len(pages), Pages: encoded}
}

// ProtectionSetting godoc
// @Summary      Set a document's protection level
// @Description  FR-6. An absent `protection_level` clears the override and puts the document back on its space's default — inheriting is the absence of a setting, not a third value.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                                 true  "Document ID"
// @Param        request  body      protection.SetProtectionLevelRequest   true  "Level"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.DocumentDetailResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /documents/{id}/protection-setting [put]
func (h *Handler) ProtectionSetting(c *gin.Context) error {
	var request protection.SetProtectionLevelRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	doc, err := h.service.SetProtectionLevel(c.Request.Context(), c.Param("id"), request.ProtectionLevel)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Protection level updated", h.detail(c, doc))
	return nil
}

// AccessLog godoc
// @Summary      Read one document's access trail
// @Description  FR-4: every view, download, print and PDF export of this document, newest first. Append-only — there is no endpoint that edits or removes a row.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id        path      string  true   "Document ID"
// @Param        page      query     int     false  "Page number"
// @Param        per_page  query     int     false  "Rows per page"
// @Param        action    query     string  false  "Filter by action"  Enums(view, download, print, export_pdf)
// @Param        from      query     string  false  "On or after (2006-01-02)"
// @Param        to        query     string  false  "On or before (2006-01-02)"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]protection.AccessLogResource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /documents/{id}/access-log [get]
func (h *Handler) AccessLog(c *gin.Context) error {
	filter, err := protection.ParseAccessLogFilter(c)
	if err != nil {
		return err
	}

	logs, meta, err := h.service.AccessLogs(c, c.Param("id"), filter)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Access log retrieved", utils.Payload{
		Data: protection.NewAccessLogResourceCollection(logs),
		Meta: meta,
	})
	return nil
}

// UserAccessLog godoc
// @Summary      Read one person's access trail
// @Description  FR-4 from the other side: everything this account opened, exported or printed. The same rows as the per-document log, filtered by reader instead of by document.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id        path      string  true   "User ID"
// @Param        page      query     int     false  "Page number"
// @Param        per_page  query     int     false  "Rows per page"
// @Param        action    query     string  false  "Filter by action"  Enums(view, download, print, export_pdf)
// @Param        from      query     string  false  "On or after (2006-01-02)"
// @Param        to        query     string  false  "On or before (2006-01-02)"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]protection.AccessLogResource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /users/{id}/access-log [get]
func (h *Handler) UserAccessLog(c *gin.Context) error {
	filter, err := protection.ParseAccessLogFilter(c)
	if err != nil {
		return err
	}

	logs, meta, err := h.service.UserAccessLogs(c, c.Param("id"), filter)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Access log retrieved", utils.Payload{
		Data: protection.NewAccessLogResourceCollection(logs),
		Meta: meta,
	})
	return nil
}

// ReportAccess godoc
// @Summary      Report a print
// @Description  A browser print never reaches the API, so the client reports it. Only `print` is accepted — every other action is observed by the server where it happens, and accepting them here would let any client write rows for things that never occurred.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                          true  "Document ID"
// @Param        request  body      protection.ReportAccessRequest  true  "Action"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /documents/{id}/access-log [post]
func (h *Handler) ReportAccess(c *gin.Context) error {
	var request protection.ReportAccessRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	authUser := c.MustGet("user").(*user.User)
	if err := h.service.ReportAccess(c, c.Param("id"), authUser, request.Action); err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Access recorded", nil)
	return nil
}

// ViewSessions godoc
// @Summary      List the live share links for a document
// @Description  FR-5. Expired links are left out: the list answers "who can open this right now". No token is returned — it was shown once, at creation.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]protection.ViewSessionResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /documents/{id}/view-sessions [get]
func (h *Handler) ViewSessions(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	sessions, err := h.service.ViewSessions(c.Request.Context(), c.Param("id"), authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "View links retrieved", protection.NewViewSessionResourceCollection(sessions))
	return nil
}

// CreateViewSession godoc
// @Summary      Create an expiring share link
// @Description  FR-5: a link to one published document, valid until it expires, usable without an account. The token is in this response and nowhere else — the row stores only its hash, so a lost link is revoked and replaced rather than looked up.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                                true  "Document ID"
// @Param        request  body      protection.CreateViewSessionRequest   true  "Link"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=protection.CreatedViewSessionResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /documents/{id}/view-sessions [post]
func (h *Handler) CreateViewSession(c *gin.Context) error {
	var request protection.CreateViewSessionRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	authUser := c.MustGet("user").(*user.User)
	session, token, err := h.service.CreateViewSession(c.Request.Context(), c.Param("id"), authUser, request.Label, request.TTL())
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "View link created", protection.NewCreatedViewSessionResource(session, token))
	return nil
}

// RevokeViewSession godoc
// @Summary      Revoke a share link early
// @Description  FR-5. Keyed on the link's id rather than its token — only the hash is stored, so nothing can find a row from a token nobody kept.
// @Tags         documents
// @Produce      json
// @Security     BearerAuth
// @Param        id         path      string  true  "Document ID"
// @Param        sessionId  path      string  true  "View link ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/view-sessions/{sessionId} [delete]
func (h *Handler) RevokeViewSession(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	if err := h.service.RevokeViewSession(c.Request.Context(), c.Param("id"), c.Param("sessionId"), authUser); err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "View link revoked", nil)
	return nil
}

// OpenViewSession godoc
// @Summary      Open a shared document by link
// @Description  FR-5, and the only route reachable without an account. Everything comes from the token: which document, for how long, and whose name the watermark carries — the person who created the link, since the recipient has none. A strict document arrives as page images here too.
// @Tags         documents
// @Produce      json
// @Param        token  path      string  true  "View link token"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=document.SharedDocumentResource}}
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /view-session/{token} [get]
func (h *Handler) OpenViewSession(c *gin.Context) error {
	shared, err := h.service.OpenViewSession(c, strings.TrimSpace(c.Param("token")))
	if err != nil {
		return err
	}

	c.Header("Cache-Control", "private, no-store")
	utils.ResponseJSON(c, enums.SUCCESS, "Shared document retrieved", NewSharedDocumentResource(shared))
	return nil
}

// SharedDocumentResource is what a link recipient gets: enough to read the
// document and nothing else.
//
// No author, no tags, no versions, no lifecycle actions, no view count. They
// are not a member of the team — a share link is a copy of one document, not a
// window into the space it lives in.
type SharedDocumentResource struct {
	Title     string           `json:"title"`
	Content   Content          `json:"content"`
	Watermark string           `json:"watermark"`
	Level     protection.Level `json:"protection_level"`
	ExpiresAt string           `json:"expires_at"`
	Pages     []string         `json:"pages,omitempty"`
}

func NewSharedDocumentResource(shared *SharedDocument) SharedDocumentResource {
	level := shared.Document.EffectiveProtection()

	resource := SharedDocumentResource{
		Title:     shared.Document.Title,
		Watermark: shared.Watermark,
		Level:     level,
		ExpiresAt: utils.FormatDateTimeVal(shared.Session.ExpiresAt),
	}

	if level == protection.LevelStrict {
		// Pages instead of content, exactly as for a logged-in reader: the
		// text does not leave the server at this level, and a recipient
		// outside the organisation is the last person to make an exception
		// for.
		resource.Content = Content("{}")
		resource.Pages = NewDocumentPagesResource(shared.Pages).Pages
		return resource
	}

	resource.Content = shared.Document.Content
	return resource
}
