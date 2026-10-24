package middleware

import (
	stderrors "errors"
	"api-kms/errors"

	"github.com/iqbalatma/gofortify"

	"github.com/gin-gonic/gin"
)

func RefreshMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mobile has no httpOnly cookie to send the refresh token back in, so it
		// carries it as a bearer token instead.
		token, _ := c.Cookie("refresh_token")
		if token == "" {
			token = c.GetHeader("Authorization")
			gofortify.RemoveBearer(&token)
		}
		payload, err := gofortify.ValidateRefreshToken(
			&token,
		)

		if err != nil {
			// stderrors.Is rather than ==, so a wrapped sentinel still matches.
			switch {
			case stderrors.Is(err, gofortify.ErrInvalidTokenType):
				abortWithError(c, errors.InvalidTokenTypeException().WithCause(err))
			default:
				abortWithError(c, errors.InvalidCredential().WithCause(err))
			}
			return
		}

		// The refresh handler only re-encodes the user, so roles are not needed here.
		authUser, httpErr := loadTokenSubject(c, payload.SUB)
		if httpErr != nil {
			abortWithError(c, httpErr)
			return
		}

		c.Set("user", authUser)
	}
}
