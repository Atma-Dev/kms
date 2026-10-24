package attachment

import "api-kms/utils"

type AttachmentResource struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	FileSize int64  `json:"file_size"`
	// URL is the only way to reach the bytes. file_path is never serialised:
	// the store is not a static mount, and a path would suggest it is.
	URL        string `json:"url"`
	UploadedBy string `json:"uploaded_by"`
	CreatedAt  string `json:"created_at"`
}

func NewAttachmentResource(a *Attachment) AttachmentResource {
	return AttachmentResource{
		ID:         a.ID.String(),
		FileName:   a.FileName,
		FileType:   a.FileType,
		FileSize:   a.FileSize,
		URL:        "/api/attachments/" + a.ID.String() + "/download",
		UploadedBy: a.UploadedBy.String(),
		CreatedAt:  utils.FormatDateTimeVal(a.CreatedAt),
	}
}

func NewAttachmentResourceCollection(attachments []Attachment) []AttachmentResource {
	result := make([]AttachmentResource, len(attachments))
	for i := range attachments {
		result[i] = NewAttachmentResource(&attachments[i])
	}
	return result
}

// InlineImageResource is the editor's answer: a URL to embed, nothing else.
type InlineImageResource struct {
	URL string `json:"url"`
}

func NewInlineImageResource(a *Attachment) InlineImageResource {
	return InlineImageResource{URL: "/api/attachments/" + a.ID.String() + "/download"}
}
