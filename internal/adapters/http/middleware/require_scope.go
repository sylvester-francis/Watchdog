package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// RequireScope returns middleware that allows the request through only when
// the upstream auth middleware has set token_scope to the expected value.
// Sessions and missing scopes are rejected — this guards endpoints that must
// only ever be reached with a specific token kind (e.g. /v1/traces with the
// telemetry_ingest scope).
func RequireScope(scope domain.TokenScope) echo.MiddlewareFunc {
	want := string(scope)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			got, _ := c.Get("token_scope").(string)
			if got != want {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "token scope insufficient for this endpoint",
				})
			}
			return next(c)
		}
	}
}
