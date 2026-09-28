package attachment

import (
	"api-kms/config"
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// fileHeader builds a real multipart part, because that is what the handler
// hands store() — constructing a FileHeader by hand would test a shape the
// code never actually receives.
func fileHeader(t *testing.T, name string, content []byte, declaredType string) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="file"; filename="` + name + `"`},
		"Content-Type":        {declaredType},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	form, err := multipart.NewReader(&body, writer.Boundary()).ReadForm(int64(len(content)) + 1024)
	if err != nil {
		t.Fatal(err)
	}
	return form.File["file"][0]
}

func withStorage(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	config.AppConfig = &config.Config{StoragePath: root}
	return root
}

const pdfBytes = "%PDF-1.4\nnot really a pdf but it sniffs as one\n"

func TestStoreWritesUnderTheSpaceFolderWithAUUIDName(t *testing.T) {
	root := withStorage(t)
	space, id := uuid.New(), uuid.New()

	relative, contentType, err := store(
		fileHeader(t, "Panduan Deploy.pdf", []byte(pdfBytes), "application/pdf"),
		space, KindAttachment, id, allowedAttachmentTypes, 10<<20,
	)
	if err != nil {
		t.Fatalf("store failed: %v", err)
	}

	if contentType != "application/pdf" {
		t.Errorf("content type = %q, want application/pdf", contentType)
	}
	want := "spaces/" + space.String() + "/attachments/" + id.String() + ".pdf"
	if relative != want {
		t.Errorf("path = %q, want %q", relative, want)
	}

	written, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("file was not written: %v", err)
	}
	if string(written) != pdfBytes {
		t.Error("stored bytes differ from what was uploaded")
	}
}

// The browser's Content-Type is a claim by the uploader. A .pdf holding HTML
// would otherwise be served back as a PDF and rendered as a page.
func TestStoreTrustsTheBytesNotTheDeclaredType(t *testing.T) {
	withStorage(t)

	_, _, err := store(
		fileHeader(t, "invoice.pdf", []byte("<html><body>hi</body></html>"), "application/pdf"),
		uuid.New(), KindAttachment, uuid.New(), allowedAttachmentTypes, 10<<20,
	)
	if err == nil {
		t.Fatal("HTML disguised as a PDF was accepted")
	}
	if !strings.Contains(err.Error(), "text/html") {
		t.Errorf("the refusal should name what it actually is, got: %v", err)
	}
}

func TestStoreRefusesAFileOverTheCap(t *testing.T) {
	root := withStorage(t)
	id := uuid.New()

	const cap = 1024
	_, _, err := store(
		fileHeader(t, "big.pdf", []byte(pdfBytes+strings.Repeat("x", cap)), "application/pdf"),
		uuid.New(), KindAttachment, id, allowedAttachmentTypes, cap,
	)
	if err == nil {
		t.Fatal("an oversized file was accepted")
	}

	// Nothing may be left behind: a partial file with no row is litter that
	// nothing will ever find again.
	var found []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			found = append(found, path)
		}
		return nil
	})
	if len(found) != 0 {
		t.Errorf("a rejected upload left files behind: %v", found)
	}
}

// The inline list is narrower than the attachment list on purpose: these are
// decoded by every reader's browser on every view.
func TestInlineImagesTakeOnlyRenderableTypes(t *testing.T) {
	withStorage(t)

	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)
	if _, _, err := store(fileHeader(t, "d.png", png, "image/png"),
		uuid.New(), KindInline, uuid.New(), allowedImageTypes, 5<<20); err != nil {
		t.Errorf("a PNG was refused as an inline image: %v", err)
	}

	if _, _, err := store(fileHeader(t, "d.pdf", []byte(pdfBytes), "application/pdf"),
		uuid.New(), KindInline, uuid.New(), allowedImageTypes, 5<<20); err == nil {
		t.Error("a PDF was accepted as an inline image")
	}
}

// storedName drops the uploaded name, so two uploads of "report.pdf" cannot
// collide — and O_EXCL means a collision would fail loudly rather than
// overwrite someone else's file.
func TestStoreRefusesToOverwriteAnExistingFile(t *testing.T) {
	withStorage(t)
	space, id := uuid.New(), uuid.New()

	if _, _, err := store(fileHeader(t, "a.pdf", []byte(pdfBytes), "application/pdf"),
		space, KindAttachment, id, allowedAttachmentTypes, 10<<20); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store(fileHeader(t, "a.pdf", []byte(pdfBytes), "application/pdf"),
		space, KindAttachment, id, allowedAttachmentTypes, 10<<20); err == nil {
		t.Error("the second write silently overwrote the first")
	}
}
