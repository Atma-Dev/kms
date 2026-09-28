package activitylog

import (
	"strings"
	"testing"
)

func TestSanitizeBodyRedactsSensitiveFields(t *testing.T) {
	body := []byte(`{"email":"a@b.com","password":"hunter2","nested":{"token":"abc123"}}`)
	got := SanitizeBody(body, "application/json")
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if strings.Contains(*got, "hunter2") || strings.Contains(*got, "abc123") {
		t.Fatalf("secret leaked into stored body: %s", *got)
	}
	if !strings.Contains(*got, "a@b.com") {
		t.Fatalf("non-sensitive field was redacted: %s", *got)
	}
}

func TestSanitizeBodyKeepsResponseEnvelopeCode(t *testing.T) {
	body := []byte(`{"code":"SUCCESS","message":"ok","status_code":200}`)
	got := SanitizeBody(body, "application/json")
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if !strings.Contains(*got, `"code":"SUCCESS"`) {
		t.Fatalf("response envelope's code field was redacted: %s", *got)
	}
}

func TestSanitizeBodySkipsNonTextContentType(t *testing.T) {
	if got := SanitizeBody([]byte("binarydata"), "application/octet-stream"); got != nil {
		t.Fatalf("expected nil for non-text content type, got %q", *got)
	}
}

func TestSanitizeBodySkipsEmpty(t *testing.T) {
	if got := SanitizeBody(nil, "application/json"); got != nil {
		t.Fatalf("expected nil for empty body, got %q", *got)
	}
}

func TestSanitizeBodyTruncatesLongBody(t *testing.T) {
	long := strings.Repeat("a", maxStoredBodyLen+100)
	got := SanitizeBody([]byte(`{"data":"`+long+`"}`), "application/json")
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if !strings.HasSuffix(*got, "...(truncated)") {
		t.Fatalf("expected truncation marker, got tail: %s", (*got)[len(*got)-30:])
	}
}
