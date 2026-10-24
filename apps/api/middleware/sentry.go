package middleware

import (
	"api-kms/app/user"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// SentryHub returns the per-request hub sentrygin bound to c (carrying
// method, URL and headers), tagged with the authenticated user when
// AuthMiddleware has set one. Falls back to the global hub if sentrygin
// wasn't installed, so a handler exercised without the full middleware
// chain (e.g. a test) still gets a safe no-op hub rather than a nil one.
func SentryHub(c *gin.Context) *sentry.Hub {
	hub := sentrygin.GetHubFromContext(c)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	if authUser, ok := c.Get("user"); ok {
		if u, ok := authUser.(*user.User); ok {
			hub.Scope().SetUser(sentry.User{
				ID:    u.ID.String(),
				Email: u.Email,
			})
		}
	}

	return hub
}
