package middleware

import (
	"api-kms/app/user"

	"github.com/gin-gonic/gin"
	goaudit "github.com/iqbalatma/go-audit"
)

// AuditContextMiddleware attaches actor + request info to the request
// context so any app/*Action deep in a service can call goaudit.Init(ctx, ...)
// without threading actor/request data through every function signature.
// Must run after AuthMiddleware, which is what puts "user" in gin's context.
func AuditContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := goaudit.WithRequestInfo(c.Request.Context(), goaudit.RequestInfoFromHTTP(c.Request))

		if value, exists := c.Get("user"); exists {
			if authUser, ok := value.(*user.User); ok {
				name := authUser.FirstName
				if authUser.LastName != nil {
					name += " " + *authUser.LastName
				}
				ctx = goaudit.WithActor(ctx, goaudit.Actor{
					Table: "users",
					ID:    authUser.ID.String(),
					Name:  name,
					Email: authUser.Email,
				})
			}
		}

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
