package attachment

import (
	"api-kms/config"
	"api-kms/errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// allowedAttachmentTypes is FR-7's "PDF, gambar, dokumen office, dll" as an
// allowlist rather than a denylist. A denylist is a list of the formats
// someone already thought of, and the one that matters is always the next one.
var allowedAttachmentTypes = map[string]bool{
	"application/pdf":  true,
	"image/jpeg":       true,
	"image/png":        true,
	"image/webp":       true,
	"image/gif":        true,
	"text/plain":       true,
	"text/csv":         true,
	"application/zip":  true,
	"application/json": true,

	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                             true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
}

// allowedImageTypes is FR-8's narrower set: these are decoded by every
// reader's browser on every view, so the list is only what renders inline.
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// sniffLength is what http.DetectContentType reads. The browser-supplied
// Content-Type is a claim by the uploader, so the stored type comes from the
// bytes instead — a .pdf that is actually an HTML page would otherwise be
// served back as a PDF and rendered as a page.
const sniffLength = 512

// store writes one uploaded file and returns its path relative to the storage
// root, plus the type sniffed from its contents.
func store(file *multipart.FileHeader, spaceID uuid.UUID, kind Kind, id uuid.UUID, allowed map[string]bool, maxBytes int64) (string, string, error) {
	if file.Size > maxBytes {
		return "", "", errors.InvalidStatusException("file is larger than the " + humanMB(maxBytes) + " limit").
			WithErrors(map[string][]string{"file": {"larger than " + humanMB(maxBytes)}})
	}

	src, err := file.Open()
	if err != nil {
		return "", "", errors.InternalServerError("failed to read uploaded file").WithCause(err)
	}
	defer src.Close()

	contentType, err := sniff(src)
	if err != nil {
		return "", "", err
	}
	if !allowed[contentType] {
		return "", "", errors.InvalidStatusException("files of type " + contentType + " are not allowed").
			WithErrors(map[string][]string{"file": {contentType + " is not an allowed file type"}})
	}

	relative := relativePath(spaceID, kind, storedName(id, file.Filename))
	absolute := filepath.Join(config.AppConfig.StoragePath, filepath.FromSlash(relative))

	if err := os.MkdirAll(filepath.Dir(absolute), 0o750); err != nil {
		return "", "", errors.InternalServerError("failed to prepare storage folder").WithCause(err)
	}

	// O_EXCL: the name is a fresh uuid, so an existing file means something is
	// very wrong and silently overwriting it would destroy someone's upload.
	dst, err := os.OpenFile(absolute, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", "", errors.InternalServerError("failed to open destination file").WithCause(err)
	}
	defer dst.Close()

	// LimitReader, not the declared size: Size is a header, and a body that
	// keeps going would otherwise keep being written.
	written, err := io.Copy(dst, io.LimitReader(src, maxBytes+1))
	if err != nil {
		_ = os.Remove(absolute)
		return "", "", errors.InternalServerError("failed to write uploaded file").WithCause(err)
	}
	if written > maxBytes {
		_ = os.Remove(absolute)
		return "", "", errors.InvalidStatusException("file is larger than the " + humanMB(maxBytes) + " limit")
	}

	return relative, contentType, nil
}

// sniff reads the leading bytes, identifies the type from them, and rewinds so
// the copy that follows still starts at byte zero.
func sniff(src multipart.File) (string, error) {
	head := make([]byte, sniffLength)
	n, err := src.Read(head)
	if err != nil && err != io.EOF {
		return "", errors.InternalServerError("failed to inspect uploaded file").WithCause(err)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", errors.InternalServerError("failed to rewind uploaded file").WithCause(err)
	}

	// DetectContentType appends "; charset=utf-8" for text; the allowlist is
	// keyed on the media type alone.
	return strings.TrimSpace(strings.Split(http.DetectContentType(head[:n]), ";")[0]), nil
}

// absolutePath resolves a stored path against the storage root and refuses
// anything that escapes it.
//
// file_path is written by this package and never by a client, so this should
// be unreachable — which is exactly why it is cheap to keep: the day it stops
// being true, it fails closed instead of serving /etc/passwd.
func absolutePath(relative string) (string, error) {
	root, err := filepath.Abs(config.AppConfig.StoragePath)
	if err != nil {
		return "", errors.InternalServerError("failed to resolve storage root").WithCause(err)
	}

	resolved, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(path.Clean("/"+relative))))
	if err != nil {
		return "", errors.InternalServerError("failed to resolve file path").WithCause(err)
	}
	if resolved != root && !strings.HasPrefix(resolved, root+string(os.PathSeparator)) {
		return "", errors.InternalServerError("stored file path escapes the storage root")
	}
	return resolved, nil
}

func remove(relative string) error {
	absolute, err := absolutePath(relative)
	if err != nil {
		return err
	}
	// A missing file is the state the caller wanted; only a real failure is an
	// error, or a half-finished earlier delete would be permanently un-retryable.
	if err := os.Remove(absolute); err != nil && !os.IsNotExist(err) {
		return errors.InternalServerError("failed to delete stored file").WithCause(err)
	}
	return nil
}

func humanMB(bytes int64) string {
	return strconv.FormatInt(bytes/(1024*1024), 10) + "MB"
}
