package attachment

import (
	"api-kms/config"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestStoredNameNeverKeepsTheUploadedName(t *testing.T) {
	id := uuid.New()

	cases := map[string]string{
		"report.pdf":                       ".pdf",
		"../../../etc/passwd":              "", // path.Ext finds nothing to keep
		"archive.tar.gz":                   ".gz",
		"UPPER.PNG":                        ".png",
		"no-extension":                     "",
		"weird." + strings.Repeat("x", 40): "", // not an extension, dropped
	}

	for input, wantExt := range cases {
		got := storedName(id, input)
		if got != id.String()+wantExt {
			t.Errorf("storedName(%q) = %q, want %q", input, got, id.String()+wantExt)
		}
		// Whatever was uploaded, the name written to disk is one path element.
		if strings.ContainsAny(got, `/\`) {
			t.Errorf("storedName(%q) produced a path separator: %q", input, got)
		}
	}
}

func TestRelativePathIsScopedToTheSpace(t *testing.T) {
	space := uuid.New()

	attachment := relativePath(space, KindAttachment, "f.pdf")
	inline := relativePath(space, KindInline, "f.png")

	if !strings.HasPrefix(attachment, "spaces/"+space.String()+"/attachments/") {
		t.Errorf("attachment path = %q", attachment)
	}
	if !strings.HasPrefix(inline, "spaces/"+space.String()+"/inline/") {
		t.Errorf("inline path = %q", inline)
	}
}

func TestAbsolutePathRefusesEscapes(t *testing.T) {
	config.AppConfig = &config.Config{StoragePath: t.TempDir()}
	root, _ := filepath.Abs(config.AppConfig.StoragePath)

	ok, err := absolutePath("spaces/abc/attachments/f.pdf")
	if err != nil {
		t.Fatalf("a normal path was refused: %v", err)
	}
	if !strings.HasPrefix(ok, root) {
		t.Errorf("resolved outside the root: %q", ok)
	}

	// file_path is written by this package, so these should be unreachable —
	// the point is that they fail closed if that ever stops being true.
	for _, escape := range []string{"../../etc/passwd", "spaces/../../../etc/passwd", "/etc/passwd"} {
		resolved, err := absolutePath(escape)
		if err == nil && !strings.HasPrefix(resolved, root) {
			t.Errorf("absolutePath(%q) escaped the root: %q", escape, resolved)
		}
	}
}
