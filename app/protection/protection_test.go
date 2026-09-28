package protection

import (
	"testing"
	"time"
)

func level(l Level) *Level { return &l }

func TestResolvePrefersTheDocumentThenTheSpace(t *testing.T) {
	cases := []struct {
		name     string
		space    Level
		document *Level
		want     Level
	}{
		{"document overrides its space", LevelNormal, level(LevelStrict), LevelStrict},
		{"a document can also relax it", LevelStrict, level(LevelNormal), LevelNormal},
		{"no override inherits the space", LevelStrict, nil, LevelStrict},
		{"neither set falls back to normal", "", nil, LevelNormal},
		// A value no version of this code writes — a hand-edited row, or a
		// level some later migration removed. Falling back to normal keeps
		// documents readable; falling back to strict would turn every space
		// into page images on a bad deploy.
		{"an unknown space level falls back to normal", Level("paranoid"), nil, LevelNormal},
		{"an unknown document level defers to the space", LevelStrict, level(Level("paranoid")), LevelStrict},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.space, c.document); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// The trail's one client-writable action. If this ever returns true for
// anything else, a client can write rows describing accesses that never
// happened, and the log stops being evidence.
func TestOnlyPrintIsClientReportable(t *testing.T) {
	for _, action := range []Action{ActionView, ActionDownload, ActionExportPDF} {
		if action.ClientReportable() {
			t.Errorf("%q must be recorded by the server, not reported by the client", action)
		}
	}
	if !ActionPrint.ClientReportable() {
		t.Error("print is the one action the server cannot observe")
	}
}

func TestHashTokenIsStableAndNotThePlaintext(t *testing.T) {
	token := "b3f0c2a1"

	first, second := HashToken(token), HashToken(token)
	if first != second {
		t.Error("the same token must hash to the same value, or no link would ever resolve")
	}
	if first == token {
		t.Error("the stored value must not be the token itself")
	}
	if len(first) != 64 {
		t.Errorf("expected a 64-character sha256 hex digest, got %d characters", len(first))
	}
	if HashToken("b3f0c2a2") == first {
		t.Error("different tokens must hash differently")
	}
}

func TestExpiredIsInclusiveOfTheExpiryMoment(t *testing.T) {
	expires := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	session := ViewSession{ExpiresAt: expires}

	if session.Expired(expires.Add(-time.Second)) {
		t.Error("a link one second before its expiry is still live")
	}
	if !session.Expired(expires.Add(time.Second)) {
		t.Error("a link one second after its expiry must be dead")
	}
}

func TestCreatorEmailIsEmptyRatherThanAPanic(t *testing.T) {
	// The watermark calls this on whatever the repository loaded. A session
	// read without its creator must render a page with no name on it, not
	// bring the request down.
	session := &ViewSession{}
	if session.CreatorEmail() != "" {
		t.Error("a session with no creator loaded should report no email")
	}
}
