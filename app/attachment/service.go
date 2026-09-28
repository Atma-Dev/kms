package attachment

import (
	"api-kms/app/document"
	"api-kms/app/protection"
	"api-kms/app/user"
	"api-kms/config"
	"api-kms/errors"
	"context"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goaudit "github.com/iqbalatma/go-audit"
)

type Service struct {
	repository *Repository
	// documents answers two questions this package must not guess at: does the
	// caller may see this document, and may they change it. Both turn on the
	// document row, which only app/document knows how to read.
	documents *document.Service
}

func (s *Service) GetByDocument(ctx context.Context, documentID string, viewer *user.User) ([]Attachment, error) {
	// Loading the document first is the access check: a restricted document
	// the caller is not named on answers 404, and its file list goes with it.
	if _, err := s.documents.GetById(ctx, documentID, viewer); err != nil {
		return nil, err
	}
	return s.repository.GetByDocument(ctx, documentID)
}

// RecordDownload notes that a file attached to a document left the server
// (Modul 4 FR-4). Delegated to app/document rather than reaching for
// app/protection directly, so every row in the trail is written through one
// place and carries the same fields.
func (s *Service) RecordDownload(c *gin.Context, documentID uuid.UUID, viewer *user.User) {
	s.documents.RecordAccess(c, documentID, viewer, protection.ActionDownload)
}

// Upload attaches a file to a document (FR-7). Answering to the document's
// edit rule rather than a permission of its own: adding a file changes what
// the document says, which is what editing means.
func (s *Service) Upload(ctx context.Context, documentID string, file *multipart.FileHeader, uploader *user.User) (*Attachment, error) {
	doc, err := s.documents.GetForEdit(ctx, documentID, uploader)
	if err != nil {
		return nil, err
	}

	a := Attachment{
		DocumentID: &doc.ID,
		Kind:       KindAttachment,
		FileName:   file.Filename,
		FileSize:   file.Size,
		UploadedBy: uploader.ID,
	}
	a.GenerateUUID()

	maxBytes := int64(config.AppConfig.UploadMaxSizeMB) * 1024 * 1024
	relative, contentType, err := store(file, doc.SpaceID, KindAttachment, a.ID, allowedAttachmentTypes, maxBytes)
	if err != nil {
		return nil, err
	}

	a.FilePath = relative
	a.FileType = contentType

	if err := s.repository.AddNewData(ctx, &a); err != nil {
		// The row is what makes the file reachable, so a file with no row is
		// invisible litter — remove it rather than leave the store growing.
		_ = remove(relative)
		return nil, err
	}

	goaudit.Init(ctx, "UPLOAD_ATTACHMENT", "upload attachment via attachment.Service.Upload").
		SetEntryObject("documents", doc.ID.String()).
		AddSingleTrail("attachments", a.ID.String(), nil, map[string]any{
			"file_name": a.FileName, "file_type": a.FileType, "file_size": a.FileSize,
		}, nil, nil).
		Execute(ctx)

	return &a, nil
}

// UploadInlineImage backs the editor's drag-and-drop (FR-8). It is a separate
// endpoint from Upload because the answer is a URL to embed, not a row in the
// attachment list, and the size and type limits are different.
//
// The image is stored against the space, not the document: the editor needs
// the URL while the document is still being written, and often before it has
// been saved at all.
func (s *Service) UploadInlineImage(ctx context.Context, spaceID string, file *multipart.FileHeader, uploader *user.User) (*Attachment, error) {
	space, err := s.documents.SpaceIDFor(ctx, spaceID)
	if err != nil {
		return nil, err
	}

	a := Attachment{Kind: KindInline, FileName: file.Filename, FileSize: file.Size, UploadedBy: uploader.ID}
	a.GenerateUUID()

	maxBytes := int64(config.AppConfig.UploadImageMaxSizeMB) * 1024 * 1024
	relative, contentType, err := store(file, space, KindInline, a.ID, allowedImageTypes, maxBytes)
	if err != nil {
		return nil, err
	}

	a.FilePath = relative
	a.FileType = contentType

	if err := s.repository.AddNewData(ctx, &a); err != nil {
		_ = remove(relative)
		return nil, err
	}

	return &a, nil
}

// OpenForDownload resolves an attachment to a path on disk, after checking the
// caller may see the document it belongs to.
func (s *Service) OpenForDownload(ctx context.Context, id string, viewer *user.User) (*Attachment, string, error) {
	a, err := s.repository.GetById(ctx, id)
	if err != nil {
		return nil, "", err
	}

	if a.DocumentID != nil {
		if _, err := s.documents.GetById(ctx, a.DocumentID.String(), viewer); err != nil {
			return nil, "", err
		}
	}

	absolute, err := absolutePath(a.FilePath)
	if err != nil {
		return nil, "", err
	}
	return a, absolute, nil
}

func (s *Service) DeleteById(ctx context.Context, id string, actor *user.User) error {
	a, err := s.repository.GetById(ctx, id)
	if err != nil {
		return err
	}
	// An inline image that never reached a document has no row to check edit
	// rights against, so the rule is who uploaded it. Refusing outright — as
	// this did — left every abandoned upload unremovable through the API, for
	// anyone, forever.
	if a.DocumentID == nil {
		if a.UploadedBy != actor.ID {
			return errors.UnauthorizedException("you may only delete your own uploads")
		}
	} else if _, err := s.documents.GetForEdit(ctx, a.DocumentID.String(), actor); err != nil {
		return err
	}

	// Row first: a row with no file 404s on download, which is recoverable. A
	// file with no row is litter nothing will ever find again.
	if err := s.repository.DeleteById(ctx, id); err != nil {
		return err
	}
	if err := remove(a.FilePath); err != nil {
		return err
	}

	// An orphan has no document to file the trail under, so it is recorded
	// against the attachment itself.
	entryID := id
	if a.DocumentID != nil {
		entryID = a.DocumentID.String()
	}
	goaudit.Init(ctx, "DELETE_ATTACHMENT", "delete attachment via attachment.Service.DeleteById").
		SetEntryObject("documents", entryID).
		AddSingleTrail("attachments", id, map[string]any{"file_name": a.FileName}, nil, nil, nil).
		Execute(ctx)

	return nil
}

func NewService(repository *Repository, documents *document.Service) *Service {
	return &Service{repository: repository, documents: documents}
}
