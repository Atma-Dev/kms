package activitylog

import (
	"regexp"
	"strings"
)

// MaxBodyCaptureBytes caps how much of a request/response body
// middleware.LoggerMiddleware buffers before SanitizeBody trims it further.
// Keeps memory bounded for large payloads (CSV exports, big list responses)
// without needing to know Content-Length upfront.
const MaxBodyCaptureBytes = 8192

const maxStoredBodyLen = 4000

// sensitiveKeyPattern matches `"key":"value"` pairs for known-sensitive JSON
// keys so their values can be redacted before a body is stored.
//
// "code" is deliberately excluded: this app's own response envelope uses
// `code` for its status slug (e.g. "SUCCESS", "ERR_..."), so redacting it
// would blank out the field that matters most for reading an error out of
// the log. The real OAuth `code`/`code_verifier` never appear in a JSON body
// here — PKCE carries them as a callback query param and an httpOnly cookie.
//
// ponytail: denylist regex, not a JSON parser — good enough for this app's
// flat request/response shapes. Upgrade to json.Unmarshal + a recursive walk
// if a nested or array-wrapped secret ever slips through.
var sensitiveKeyPattern = regexp.MustCompile(`(?i)"(password|secret|token|access_token|refresh_token|client_secret)"\s*:\s*"[^"]*"`)

// SanitizeBody redacts known-sensitive fields and truncates the result for
// storage in activity_logs. Returns nil for an empty or non-text body —
// binary bodies (file uploads/downloads) aren't useful in an audit trail.
func SanitizeBody(body []byte, contentType string) *string {
	if len(body) == 0 {
		return nil
	}
	if !strings.Contains(contentType, "json") && !strings.Contains(contentType, "text") {
		return nil
	}

	s := string(sensitiveKeyPattern.ReplaceAll(body, []byte(`"$1":"***REDACTED***"`)))
	if len(s) > maxStoredBodyLen {
		// Byte-slicing a string can split a multi-byte rune; ToValidUTF8
		// cleans up whatever that leaves behind at the cut point.
		s = strings.ToValidUTF8(s[:maxStoredBodyLen], "") + "...(truncated)"
	}
	return &s
}
