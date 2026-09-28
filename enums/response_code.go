package enums

import "net/http"

type ResponseCode string

const (
	SUCCESS                     ResponseCode = "SUCCESS"
	CREATED                     ResponseCode = "CREATED"
	ERR_NOT_FOUND               ResponseCode = "ERR_NOT_FOUND"
	ERR_ROUTE_NOT_FOUND         ResponseCode = "ERR_ROUTE_NOT_FOUND"
	ERR_ACTION_UNAUTHORIZED     ResponseCode = "ERR_ACTION_UNAUTHORIZED"
	ERR_INVALID_ACTION          ResponseCode = "ERR_INVALID_ACTION"
	ERR_INVALID_STATUS          ResponseCode = "ERR_INVALID_STATUS"
	ERR_AUTHENTICATION          ResponseCode = "ERR_AUTHENTICATION"
	ERR_INVALID_QUERY_PARAMETER ResponseCode = "ERR_INVALID_QUERY_PARAMETER"
	ERR_INTERNAL_SERVER_ERROR   ResponseCode = "ERR_INTERNAL_SERVER_ERROR"
	ERR_BAD_REQUEST             ResponseCode = "ERR_BAD_REQUEST"
)

// Map untuk mengaitkan setiap ResponseCode dengan HTTP status code
var responseCodeHTTPStatus = map[ResponseCode]int{
	SUCCESS:                 http.StatusOK,
	CREATED:                 http.StatusCreated,
	ERR_NOT_FOUND:           http.StatusNotFound,
	ERR_ROUTE_NOT_FOUND:     http.StatusNotFound,
	ERR_ACTION_UNAUTHORIZED: http.StatusForbidden,
	ERR_AUTHENTICATION:      http.StatusUnauthorized,
	ERR_INVALID_ACTION:      http.StatusForbidden,
	ERR_INVALID_STATUS:      http.StatusUnprocessableEntity,
	ERR_BAD_REQUEST:         http.StatusBadRequest,

	ERR_INVALID_QUERY_PARAMETER: http.StatusBadRequest,
	ERR_INTERNAL_SERVER_ERROR:   http.StatusInternalServerError,
}

func (r ResponseCode) HTTPStatus() int {
	if status, exist := responseCodeHTTPStatus[r]; exist {
		return status
	}

	return http.StatusInternalServerError
}
