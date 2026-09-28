package activitylog

import (
	"net/http"
	"strings"
)

// excludedPathPrefixes are endpoints that aren't a business feature to
// audit: auth/session plumbing and the "who am I" check. Everything else —
// deposits, invoices, payments, remittances, partners, management, reports,
// notifications — is a feature and stays logged.
var excludedPathPrefixes = []string{
	"/api/auth",
	"/api/profiles/me",
}

var mutatingMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPatch:  true,
	http.MethodPut:    true,
	http.MethodDelete: true,
}

// ShouldLog reports whether a request is worth an activity_logs row. GET
// (and HEAD/OPTIONS) never change state, so they're excluded outright —
// only a request that could have changed something is worth auditing. That
// also makes /api/management/audits exclude itself for free: it's GET-only.
func ShouldLog(method, path string) bool {
	if !mutatingMethods[method] {
		return false
	}
	for _, prefix := range excludedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	return true
}
