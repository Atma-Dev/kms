package utils

import (
	"strconv"
	"strings"
	"unicode"
)

// Slugify turns a title into a URL segment: lowercase, ASCII alphanumerics
// kept, everything else collapsed to a single hyphen.
//
// Runs above ASCII are dropped rather than transliterated. A title that is
// entirely non-ASCII therefore slugs to "" — UniqueSlug is what turns that
// into something usable, so the emptiness is handled in one place instead of
// by every caller.
func Slugify(s string) string {
	var b strings.Builder
	lastHyphen := true // leading hyphens are never written

	for _, r := range strings.ToLower(s) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			lastHyphen = false
		case !lastHyphen:
			b.WriteByte('-')
			lastHyphen = true
		}
	}

	return strings.TrimSuffix(b.String(), "-")
}

// UniqueSlug returns base, or base-2, base-3, … — the first that exists
// reports false for.
//
// exists is the caller's own scoped query: slugs are unique per space, not
// globally, so only the repository knows what "taken" means here.
//
// The suffix starts at 2 so the first duplicate reads "-2" and not "-1", which
// would suggest there is a "-0" somewhere.
//
// This is advisory, not a guarantee: two requests can both read "free" before
// either writes. The unique index is what actually decides, and the repository
// translates its violation — see errors.DataAlreadyExistException.
func UniqueSlug(base string, exists func(candidate string) (bool, error)) (string, error) {
	if base == "" {
		base = "untitled"
	}

	candidate := base
	// ponytail: linear probe. Fine at a handful of collisions per title; if a
	// space ever holds thousands of "Meeting Notes", query MAX(suffix) once
	// instead.
	for i := 2; ; i++ {
		taken, err := exists(candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		candidate = base + "-" + strconv.Itoa(i)
	}
}
