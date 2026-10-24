package middleware

import (
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/errors"

	"github.com/gin-gonic/gin"
)

// hasPermission defers to user.User.HasPermission so the route gates and the
// ownership checks inside services answer from one implementation — a second
// copy here is how the two drift apart.
func hasPermission(c *gin.Context, permission enums.Permission) bool {
	return c.MustGet("user").(*user.User).HasPermission(permission)
}

// RequirePermission is §4.5d's PIC gate: "kapabilitas approve/reject ini
// tidak dimiliki Admin biasa." Must run after AuthMiddleware.
func RequirePermission(permission enums.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasPermission(c, permission) {
			return
		}
		abortWithError(c, errors.UnauthorizedException("you do not have permission to perform this action"))
	}
}

// RequireAnyPermission passes when the caller holds at least one of the
// listed permissions.
//
// It exists for grants that come in a narrow and a wide form — documents.update
// (what I authored) and documents.update.all (anyone's). Gating the route on
// the narrow one alone locks out every role that holds only the wide one,
// which is the wrong way round: they can do strictly more.
//
// The route only decides whether the caller may attempt the edit at all.
// Whether they may edit *this* document is settled per row in app/document,
// which is the only place that knows who wrote it.
func RequireAnyPermission(permissions ...enums.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, permission := range permissions {
			if hasPermission(c, permission) {
				return
			}
		}
		abortWithError(c, errors.UnauthorizedException("you do not have permission to perform this action"))
	}
}

// RequirePermissionForQueryParam gates a single filter parameter rather than a
// whole route: the request is refused only when param is actually present and
// the caller lacks permission. A caller who never sends it is unaffected, so
// the page still loads — it just cannot use that one filter.
//
// This is the server-side half of a filter dropdown's permission. The
// parameter is refused rather than quietly dropped: silently ignoring it would
// answer a request for "user X's activity" with everyone's, and the caller
// could not tell the difference. utils.positiveIntQuery already takes the same
// stance on an unusable page value.
func RequirePermissionForQueryParam(param string, permission enums.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Query(param) == "" {
			return
		}
		if hasPermission(c, permission) {
			return
		}
		// Names the parameter — a 403 on a page the caller just successfully
		// opened is otherwise indistinguishable from the page itself being
		// forbidden.
		abortWithError(c, errors.UnauthorizedException(
			"you do not have permission to filter by "+param))
	}
}
