// Package attachment stores the files hung off a document: downloadable
// attachments (FR-7) and the images embedded in its content (FR-8).
//
// Files live on local disk under config.StoragePath, laid out by space:
//
//	storage/spaces/{spaceId}/attachments/{uuid}{ext}
//	storage/spaces/{spaceId}/inline/{uuid}{ext}
//
// Per space rather than per document, because a space is the unit anyone
// actually acts on — granting, backing up or deleting. A per-document folder
// would be thousands of directories holding one file each.
//
// Nothing is served as a static folder. Every read goes through Download,
// which asks app/document whether this caller may see the document first —
// a static mount would hand the whole store to anyone who guessed a path, and
// FR-9's restricted documents would be restricted in name only.
package attachment

import (
	"api-kms/app"
	"path"
	"strings"

	"github.com/google/uuid"
)

// Kind separates the two upload paths sharing this table. An inline image is
// already visible in the document, so it never appears in the attachment list
// — but it needs a row so it can be found and cleaned up.
type Kind string

const (
	KindAttachment Kind = "attachment"
	KindInline     Kind = "inline"
)

type Attachment struct {
	app.BaseModel `gorm:"embedded"`
	// DocumentID is nullable for an inline image uploaded while the document
	// is still being written: the editor needs the URL before there is a
	// document to attach it to.
	DocumentID *uuid.UUID `json:"document_id" gorm:"column:document_id"`
	Kind       Kind       `json:"kind" gorm:"column:kind"`
	// FilePath is relative to config.StoragePath, always starting with the
	// space id — so a space's files are findable without the database.
	FilePath   string    `json:"-" gorm:"column:file_path"`
	FileName   string    `json:"file_name" gorm:"column:file_name"`
	FileType   string    `json:"file_type" gorm:"column:file_type"`
	FileSize   int64     `json:"file_size" gorm:"column:file_size"`
	UploadedBy uuid.UUID `json:"uploaded_by" gorm:"column:uploaded_by"`
}

func (Attachment) TableName() string {
	return "attachments"
}

// storedName is the name a file is written under: a fresh uuid plus the
// original extension.
//
// The uploaded name is never used on disk. It can contain "..", a path
// separator, a NUL, or 3,000 characters of unicode, and every one of those is
// a way to write outside the store. The real name is kept in the database
// column, where it is data rather than a path.
func storedName(id uuid.UUID, originalName string) string {
	ext := strings.ToLower(path.Ext(originalName))
	// An "extension" this long is not one, and a separator inside it means the
	// name was shaped to escape the folder rather than to describe a file.
	if len(ext) > 10 || strings.ContainsAny(ext, `/\`) {
		ext = ""
	}
	return id.String() + ext
}

// relativePath is where a file lives under the storage root.
func relativePath(spaceID uuid.UUID, kind Kind, storedFileName string) string {
	folder := "attachments"
	if kind == KindInline {
		folder = "inline"
	}
	return path.Join("spaces", spaceID.String(), folder, storedFileName)
}
