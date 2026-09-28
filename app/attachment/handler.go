package attachment

import (
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/errors"
	"api-kms/utils"
	"net/url"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

// Index godoc
// @Summary      List a document's attachments
// @Description  Downloadable files only — images embedded in the content are already visible there.
// @Tags         attachments
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]attachment.AttachmentResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /documents/{id}/attachments [get]
func (h *Handler) Index(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	attachments, err := h.service.GetByDocument(c.Request.Context(), c.Param("id"), authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Attachments retrieved", NewAttachmentResourceCollection(attachments))
	return nil
}

// Store godoc
// @Summary      Attach a file to a document
// @Description  multipart/form-data, field `file`. Type is taken from the file's own bytes, not the browser's Content-Type header. Answers to the document's edit rule.
// @Tags         attachments
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string  true  "Document ID"
// @Param        file  formData  file    true  "File"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=attachment.AttachmentResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /documents/{id}/attachments [post]
func (h *Handler) Store(c *gin.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return errors.InvalidStatusException("a file is required").
			WithErrors(map[string][]string{"file": {"a file is required"}}).
			WithCause(err)
	}

	authUser := c.MustGet("user").(*user.User)
	a, err := h.service.Upload(c.Request.Context(), c.Param("id"), file, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "Attachment uploaded", NewAttachmentResource(a))
	return nil
}

// StoreInlineImage godoc
// @Summary      Upload an image to embed in a document's content
// @Description  multipart/form-data, fields `file` and `space_id`. Returns a URL for the editor to embed. Separate from attachments: the answer is a URL, not a row in the file list, and the limits are tighter (FR-8).
// @Tags         attachments
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        space_id  formData  string  true  "Space ID the image is filed under"
// @Param        file      formData  file    true  "Image (jpg, png, webp, gif)"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=attachment.InlineImageResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /uploads/inline-image [post]
func (h *Handler) StoreInlineImage(c *gin.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return errors.InvalidStatusException("an image is required").
			WithErrors(map[string][]string{"file": {"an image is required"}}).
			WithCause(err)
	}

	spaceID := c.PostForm("space_id")
	if spaceID == "" {
		return errors.InvalidStatusException("space_id is required").
			WithErrors(map[string][]string{"space_id": {"space_id is required"}})
	}

	authUser := c.MustGet("user").(*user.User)
	a, err := h.service.UploadInlineImage(c.Request.Context(), spaceID, file, authUser)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "Image uploaded", NewInlineImageResource(a))
	return nil
}

// Download godoc
// @Summary      Download an attachment
// @Description  Streams the file after checking the caller may see the document it belongs to. The store is deliberately not a static folder — a static mount would hand every file to anyone who guessed a path, and FR-9's restricted documents would be restricted in name only.
// @Tags         attachments
// @Produce      octet-stream
// @Security     BearerAuth
// @Param        id   path      string  true  "Attachment ID"
// @Success      200  {file}    binary
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /attachments/{id}/download [get]
func (h *Handler) Download(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	a, absolute, err := h.service.OpenForDownload(c.Request.Context(), c.Param("id"), authUser)
	if err != nil {
		return err
	}

	// inline for images so an embedded one renders, attachment for everything
	// else so a click saves the file instead of navigating to it.
	disposition := "attachment"
	if a.Kind == KindInline {
		disposition = "inline"
	}

	// The uploaded name reaches a header here, so it is quoted and escaped:
	// a filename with a newline in it would otherwise let the uploader write
	// their own response headers.
	c.Header("Content-Disposition",
		disposition+`; filename="`+url.PathEscape(a.FileName)+`"`)
	c.Header("Content-Type", a.FileType)
	// The bytes are only reachable with a valid token, so caches must not keep
	// a copy that outlives the permission to read it.
	c.Header("Cache-Control", "private, no-store")

	// Modul 4 FR-4. Only a real attachment: an inline image is fetched by the
	// browser once per <img> on every page load, so logging those would bury
	// the trail under its own noise without recording a single decision anyone
	// made. An orphan upload has no document to log against.
	if a.Kind == KindAttachment && a.DocumentID != nil {
		h.service.RecordDownload(c, *a.DocumentID, authUser)
	}

	c.File(absolute)
	return nil
}

// Destroy godoc
// @Summary      Delete an attachment
// @Tags         attachments
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Attachment ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /attachments/{id} [delete]
func (h *Handler) Destroy(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	if err := h.service.DeleteById(c.Request.Context(), c.Param("id"), authUser); err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "Attachment deleted", nil)
	return nil
}

func NewHandler(service *Service) *Handler {
	return &Handler{service}
}
