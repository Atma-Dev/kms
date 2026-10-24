package utils

import (
	"fmt"
	"api-kms/enums"
	"time"

	"github.com/gin-gonic/gin"
)

type BaseHttpResponse struct {
	Code       enums.ResponseCode `json:"code"`
	Message    string             `json:"message"`
	StatusCode int                `json:"status_code"`
	Timestamp  time.Time          `json:"timestamp"`
	Payload    *Payload           `json:"payload"`
}
type HTTPResponse struct {
	BaseHttpResponse
}

type HTTPError struct {
	BaseHttpResponse
	Errors *map[string][]string `json:"errors"`
	// Debug carries the cause when APP_DEBUG is on. omitempty keeps production
	// responses byte identical to what they were before the flag existed.
	Debug *string `json:"debug,omitempty"`
	// cause is the error this one was built from. It is unexported on purpose:
	// it reaches the log through Error(), never the client through JSON.
	cause error
}

// WithCause attaches the underlying error so it survives into the logs while
// Message stays safe to show to the client. Never put err.Error() in Message.
func (H *HTTPError) WithCause(err error) *HTTPError {
	H.cause = err
	return H
}

// WithErrors attaches per-field messages so a constraint the database enforces
// (a unique code, say) reaches the client in the same shape as a validation
// failure from the binding layer.
func (H *HTTPError) WithErrors(fields map[string][]string) *HTTPError {
	H.Errors = &fields
	return H
}

// WithPayload attaches structured detail that the client is meant to render
// alongside the message — the deposit rejection sends the limit breakdown this
// way so the UI shows the same numbers whether the request succeeded or not.
func (H *HTTPError) WithPayload(data interface{}) *HTTPError {
	H.Payload = &Payload{Data: data}
	return H
}

// Cause exposes the wrapped error so ErrorHandler can decide whether the current
// environment is allowed to show it.
func (H *HTTPError) Cause() error {
	return H.cause
}

// Unwrap keeps errors.Is/errors.As working through the wrap, so callers can
// still match on things like gorm.ErrRecordNotFound.
func (H *HTTPError) Unwrap() error {
	return H.cause
}

func (H *HTTPError) Error() string {
	message := fmt.Sprintf(
		"message : %s, code: %s, timestamp: %s, status_code: %d",
		H.Message,
		H.Code,
		H.Timestamp,
		H.StatusCode,
	)
	if H.cause != nil {
		message += ", cause: " + H.cause.Error()
	}
	return message
}

type Payload struct {
	Data interface{}     `json:"data"`
	Meta *PaginationMeta `json:"meta,omitempty"`
}

type PaginationMeta struct {
	CurrentPage int    `json:"current_page"`
	From        int    `json:"from"`
	To          int    `json:"to"`
	LastPage    int    `json:"last_page"`
	Path        string `json:"path"`
	PerPage     int    `json:"per_page"`
	Total       int64  `json:"total"`
}

func NewHttpSuccess(message string, payload *Payload, responseCode ...enums.ResponseCode) *HTTPResponse {
	responseStatus := enums.SUCCESS
	if len(responseCode) > 0 {
		responseStatus = responseCode[0]
	}
	return &HTTPResponse{
		BaseHttpResponse: BaseHttpResponse{
			Code:       responseStatus,
			Timestamp:  time.Now(),
			StatusCode: responseStatus.HTTPStatus(),
			Message:    message,
			Payload:    payload,
		},
	}
}

func NewHttpError(message string, code enums.ResponseCode, errors *map[string][]string) *HTTPError {
	return &HTTPError{
		BaseHttpResponse: BaseHttpResponse{
			Code:       code,
			StatusCode: code.HTTPStatus(),
			Timestamp:  time.Now(),
			Message:    message,
			Payload:    nil,
		},
		Errors: errors,
	}
}

func ResponseJSON(c *gin.Context, code enums.ResponseCode, message string, data interface{}) {
	var finalPayload *Payload

	switch v := data.(type) {
	case *Payload:
		finalPayload = v
	case Payload:
		finalPayload = &v
	default:
		finalPayload = &Payload{
			Data: data,
			Meta: nil,
		}
	}

	c.JSON(
		code.HTTPStatus(),
		NewHttpSuccess(message, finalPayload, code),
	)
}
