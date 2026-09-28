package middleware

import (
	"api-kms/app/m2mclient"
	"api-kms/errors"

	"github.com/gin-gonic/gin"
)

// RequireM2MScope is RequirePermission's M2M counterpart: it must run after
// M2MAuthMiddleware, which is what puts "m2m_client" in the context. Keeps
// one client credential from reaching routes it was never issued for.
func RequireM2MScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client := c.MustGet("m2m_client").(*m2mclient.M2MClient)
		if client.Scope.Has(scope) {
			return
		}
		abortWithError(c, errors.UnauthorizedException("this client is not permitted to call this endpoint"))
	}
}
