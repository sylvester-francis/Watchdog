package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Error codes returned in API responses.
const (
	CodeValidation   = "VALIDATION_ERROR"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"
	CodeConflict     = "CONFLICT"
	CodeLimitReached = "LIMIT_REACHED"
	CodeRateLimited  = "RATE_LIMITED"
	CodeInternal     = "INTERNAL_ERROR"
)

// ErrorBody is the structured error payload.
type ErrorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []FieldError  `json:"details,omitempty"`
}

// FieldError describes a validation error on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// errorResponse wraps ErrorBody under the "error" key.
type errorResponse struct {
	Error ErrorBody `json:"error"`
}

// statusToCode maps HTTP status codes to default error codes.
func statusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return CodeValidation
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusTooManyRequests:
		return CodeRateLimited
	default:
		return CodeInternal
	}
}

// errJSON returns a structured JSON error response.
// The error code is derived from the HTTP status code.
func errJSON(c echo.Context, status int, msg string) error {
	return c.JSON(status, errorResponse{
		Error: ErrorBody{Code: statusToCode(status), Message: msg},
	})
}

// errJSONCode returns a structured JSON error response with an explicit code.
func errJSONCode(c echo.Context, status int, code, msg string) error {
	return c.JSON(status, errorResponse{
		Error: ErrorBody{Code: code, Message: msg},
	})
}

// errValidation returns a 400 error with field-level details.
func errValidation(c echo.Context, msg string, details ...FieldError) error {
	return c.JSON(http.StatusBadRequest, errorResponse{
		Error: ErrorBody{Code: CodeValidation, Message: msg, Details: details},
	})
}
