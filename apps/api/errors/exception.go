package errors

import (
	"api-kms/enums"
	"api-kms/utils"
	stderrors "errors"

	"gorm.io/gorm"
)

func newError(code enums.ResponseCode, defaultMessage string, messages ...string) *utils.HTTPError {
	message := defaultMessage
	if len(messages) > 0 && messages[0] != "" {
		message = messages[0]
	}

	return utils.NewHttpError(message, code, nil)
}

func InvalidCredential(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_AUTHENTICATION, "Invalid credential", messages...)
}

func TokenExpiredException(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_AUTHENTICATION, "token expired", messages...)
}

func InvalidAction(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_INVALID_ACTION, "Invalid action", messages...)
}

// InvalidStatusException is 422: the request is well formed, but the state it
// asks for is not reachable from the state the row is in — publishing an
// archived document, nesting a category past the depth cap. Distinct from
// InvalidAction (403), which is about who is asking rather than what they ask.
func InvalidStatusException(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_INVALID_STATUS, "Invalid status for this action", messages...)
}

func QueryParameterInvalid(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_INVALID_QUERY_PARAMETER, "Invalid query parameter", messages...)
}

func InvalidTokenTypeException(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_AUTHENTICATION, "Invalid token type", messages...)
}

func InternalServerError(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_INTERNAL_SERVER_ERROR, "Internal server error", messages...)
}

func UnauthorizedException(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_ACTION_UNAUTHORIZED, "Unauthorized", messages...)
}

func NonDeleteableException(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_BAD_REQUEST, "This entity cannot be deleted", messages...)
}

func DataAlreadyExistException(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_BAD_REQUEST, "Data already exists", messages...)
}

func EmptyDataException(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_NOT_FOUND, "This entity not found", messages...)
}

func InvalidParsingException(messages ...string) *utils.HTTPError {
	return newError(enums.ERR_BAD_REQUEST, "Invalid parsing exception", messages...)
}

// TranslateDuplicateCode turns a unique index violation on a master `code`
// column into the field error the form can render, instead of a 500. It relies
// on gorm's TranslateError (set in config.ConnectDB) to surface ErrDuplicatedKey
// from the driver error.
func TranslateDuplicateCode(err error, fallbackMessage string) error {
	if stderrors.Is(err, gorm.ErrDuplicatedKey) {
		return DataAlreadyExistException("code already exists").
			WithErrors(map[string][]string{"code": {"code already exists"}}).
			WithCause(err)
	}
	return InternalServerError(fallbackMessage).WithCause(err)
}
