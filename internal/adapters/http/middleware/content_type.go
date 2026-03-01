package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// RequireJSONContentType returns middleware that rejects POST, PUT, and PATCH
// requests to /api/v1/ endpoints that do not carry an application/json
// Content-Type header. GET, DELETE, HEAD, and OPTIONS are always allowed
// through. Multipart form-data requests are also permitted for file uploads.
// Requests with no body (Content-Length 0 or missing) are allowed through.
func RequireJSONContentType() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			method := c.Request().Method

			// Only enforce on methods that carry a request body.
			if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
				return next(c)
			}

			// Skip if there is no body (Content-Length 0 or absent).
			if c.Request().ContentLength == 0 {
				return next(c)
			}

			ct := c.Request().Header.Get(echo.HeaderContentType)

			// Allow multipart uploads (file upload endpoints).
			if strings.HasPrefix(ct, "multipart/") {
				return next(c)
			}

			// Require application/json (with optional charset/parameters).
			if !strings.HasPrefix(ct, echo.MIMEApplicationJSON) {
				return c.JSON(http.StatusUnsupportedMediaType, map[string]string{
					"error": "Content-Type must be application/json",
				})
			}

			return next(c)
		}
	}
}
