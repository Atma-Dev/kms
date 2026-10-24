package middleware

import (
	"api-kms/app/user"
	"api-kms/config"
	"api-kms/errors"
	"api-kms/utils"
	stderrors "errors"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/iqbalatma/gofortify"
	"gorm.io/gorm"
)

var (
	errMembershipInactive = stderrors.New("membership: account is not active")
	errWSOnlyToken        = stderrors.New("token is scoped to websocket routes only")
)

// abortWithError hands err to ErrorHandler, which main.go registers ahead of
// these middlewares. Rendering the response here instead would skip logError,
// which is why rejected requests used to leave no trace in go.log at all.
func abortWithError(c *gin.Context, err error) {
	c.Error(err)
	c.Abort()
}

// loadTokenSubject resolves a validated token's subject to its users row. A
// subject with no row means the account was removed while its token was still
// alive, so the token is no longer a valid credential.
//
// Checked fresh on every request (this already queries the DB per request, so
// this piggybacks on that): a JWT is stateless and has no way to reflect a
// revocation that happened after it was issued. This app has no server-side
// session row to delete, so the equivalent is refusing the token here instead
// — the account still exists but is no longer allowed in, which is the same
// "no longer a valid credential" case as a removed subject.
func loadTokenSubject(c *gin.Context, subject string, preloads ...string) (*user.User, *utils.HTTPError) {
	query := config.DB.WithContext(c.Request.Context())
	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	var userFromDB user.User
	if err := query.Where("id = ?", subject).First(&userFromDB).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.InvalidCredential().WithCause(gofortify.ErrJWTSubjectNotFound)
		}
		return nil, errors.InternalServerError("failed to load authenticated user").WithCause(err)
	}

	if userFromDB.MembershipStatus != user.MembershipStatusActive {
		return nil, errors.InvalidCredential().WithCause(errMembershipInactive)
	}

	return &userFromDB, nil
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token = c.GetHeader("Authorization")
		accessTokenVerifier, _ := c.Cookie("access_token_verifier")
		payload, err := gofortify.ValidateAccessToken(
			&token,
			&accessTokenVerifier,
		)

		if err != nil {
			// Anything about the token itself answers 401, never 403, so the SPA
			// can tell "log in again" apart from "you lack permission". Every
			// constructor below carries ERR_AUTHENTICATION.
			//
			// The validation error describes a token the caller supplied, so it
			// belongs in the cause where only the log can see it.
			switch {
			case stderrors.Is(err, gofortify.ErrInvalidTokenType):
				abortWithError(c, errors.InvalidTokenTypeException().WithCause(err))
			case stderrors.Is(err, gofortify.ErrExpiredToken), stderrors.Is(err, jwt.ErrTokenExpired):
				abortWithError(c, errors.TokenExpiredException().WithCause(err))
			default:
				abortWithError(c, errors.InvalidCredential().WithCause(err))
			}
			return
		}

		// A ws-token (POST /auth/ws-token) validates fine here — iuc=false, right
		// subject, not expired — but it was only ever meant for the WS routes.
		// This is the actual restriction; WSTokenAuthMiddleware itself trusts any
		// validated iuc=false token, ws-token or not.
		if utils.IsWSOnlyToken(c.Request.Context(), payload.JTI) {
			abortWithError(c, errors.InvalidCredential().WithCause(errWSOnlyToken))
			return
		}

		// Falling through a failed lookup used to leave a zero valued user in the
		// context, letting the request continue as nobody.
		authUser, httpErr := loadTokenSubject(c, payload.SUB, "Roles.Permissions")
		if httpErr != nil {
			abortWithError(c, httpErr)
			return
		}

		c.Set("user", authUser)
	}
}

// WSTokenAuthMiddleware authenticates a WebSocket handshake. A browser's
// WebSocket handshake can't set a custom Authorization header, so the access
// token travels as a query param instead.
//
// It does NOT read access_token_verifier from a cookie the way AuthMiddleware
// does. That was the original design — same-origin handshake, browser
// attaches the cookie automatically — but in practice a browser does not
// reliably send it when the WS host is a different subdomain from the page
// that opened the connection, which is exactly this app's deployment shape.
// So the token used here is expected to be one issued by POST /auth/ws-token
// (iuc=false, no cookie needed at all) rather than the regular access token.
// Mobile's own regular token already has iuc=false too and keeps working
// here unchanged.
//
// This middleware trusts any validated iuc=false token; it does not itself
// check that the token came from /auth/ws-token. What actually confines a
// ws-token to WS routes and nowhere else is AuthMiddleware refusing it via
// utils.IsWSOnlyToken — deliberately not duplicated here, since duplicating
// it would just be two places that have to agree instead of one.
//
// preloads is forwarded to loadTokenSubject — pass "Roles.Permissions" when
// the route chains RequirePermission afterward (e.g. the audits feed), which
// reads authUser.Roles straight off the context user with no query of its
// own.
func WSTokenAuthMiddleware(preloads ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token = c.Query("token")
		payload, err := gofortify.ValidateAccessToken(&token, nil)

		if err != nil {
			switch {
			case stderrors.Is(err, gofortify.ErrInvalidTokenType):
				abortWithError(c, errors.InvalidTokenTypeException().WithCause(err))
			case stderrors.Is(err, gofortify.ErrExpiredToken), stderrors.Is(err, jwt.ErrTokenExpired):
				abortWithError(c, errors.TokenExpiredException().WithCause(err))
			default:
				abortWithError(c, errors.InvalidCredential().WithCause(err))
			}
			return
		}

		authUser, httpErr := loadTokenSubject(c, payload.SUB, preloads...)
		if httpErr != nil {
			abortWithError(c, httpErr)
			return
		}

		c.Set("user", authUser)
	}
}

// TokenOrHeaderAuthMiddleware accepts either the ordinary Bearer header or a
// `?token=` query parameter, and is the only place that does.
//
// It exists for the one thing a header cannot reach: a browser's own fetch for
// a URL. An `<img src>` for an inline image and an `<a download>` for an
// attachment are both issued by the browser, which attaches no Authorization
// header — and serving those files from a static folder instead would hand the
// whole store to anyone who guessed a path, leaving FR-9's restricted
// documents restricted in name only.
//
// The query token is the same short-lived, WS-scoped token as
// WSTokenAuthMiddleware's, so a URL that leaks (a referer header, a pasted
// link, shell history) stops working in minutes rather than granting the
// bearer full API access.
func TokenOrHeaderAuthMiddleware(preloads ...string) gin.HandlerFunc {
	withQueryToken := WSTokenAuthMiddleware(preloads...)
	// AuthMiddleware preloads roles itself, so it takes no preload list.
	withHeader := AuthMiddleware()

	return func(c *gin.Context) {
		if c.Query("token") != "" {
			withQueryToken(c)
			return
		}
		withHeader(c)
	}
}
