package middleware

import (
	"errors"
	"fmt"
	"io"
	"api-kms/config"
	"api-kms/enums"
	"api-kms/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		ginErr := c.Errors.Last()
		if ginErr == nil {
			return
		}
		originalErr := ginErr.Err

		// The As branch has to come first. An HTTPError already carries the status
		// and the message that was chosen for it, and errors.Is now sees through
		// WithCause, so the sentinel checks below would otherwise hijack a wrapped
		// gorm.ErrRecordNotFound and replace "deposit not found" with their own
		// generic text. The sentinels are fallbacks for raw errors that never got
		// wrapped.
		var httpResponse *utils.HTTPError
		detail := originalErr
		switch {
		case errors.As(originalErr, &httpResponse):
			detail = httpResponse.Cause()
		case errors.Is(originalErr, io.EOF):
			httpResponse = utils.NewHttpError("Empty request body", enums.ERR_BAD_REQUEST, nil)
		case errors.Is(originalErr, gorm.ErrRecordNotFound):
			httpResponse = utils.NewHttpError("Data not found", enums.ERR_NOT_FOUND, nil)
		default:
			// Never originalErr.Error() in the message: that put raw driver text such
			// as "Unknown column 'canceled_by_id'" straight into the response. The
			// real error is already in go.log via logError.
			httpResponse = utils.NewHttpError("Internal server error", enums.ERR_INTERNAL_SERVER_ERROR, nil)
		}

		logError(c, originalErr, httpResponse.StatusCode)
		c.AbortWithStatusJSON(httpResponse.StatusCode, withDebug(httpResponse, detail))
	}
}

// withDebug attaches detail to the response only when APP_DEBUG is on. This is
// the single gate for showing internals to a client; nothing outside this file
// should consult config.DebugEnabled.
func withDebug(httpErr *utils.HTTPError, detail error) *utils.HTTPError {
	if detail == nil || !config.DebugEnabled() {
		return httpErr
	}

	text := detail.Error()
	httpErr.Debug = &text
	return httpErr
}

// logError keeps 4xx out of the red. A client mistake is not a server error, so
// it logs at warn level in yellow; only 5xx gets Error/red. Both reset the colour
// so the next line printed is not red too.
func logError(c *gin.Context, err error, status int) {
	if status >= http.StatusInternalServerError {
		Logger(c).Error(err)
		fmt.Printf(utils.ANSI_RED+"Error : %s. Error type: %T\n"+utils.ANSI_RESET, err, err)
		SentryHub(c).CaptureException(err)
		return
	}

	Logger(c).Warn(err)
	fmt.Printf(utils.ANSI_YELLOW+"Warn : %s. Error type: %T\n"+utils.ANSI_RESET, err, err)
}
