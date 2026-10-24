package utils

import "github.com/gin-gonic/gin"

func Deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// IsWebClient tells cookie-based auth (httpOnly verifier/refresh cookies) apart
// from bearer-only auth (tokens returned in the response body). Mobile can't
// rely on httpOnly cookies surviving across app restarts / native HTTP clients,
// so it opts in to bearer-only via this header; anything else defaults to web
// to keep the existing browser flow working unchanged.
func IsWebClient(c *gin.Context) bool {
	return c.GetHeader("X-Client-Type") != "mobile"
}
