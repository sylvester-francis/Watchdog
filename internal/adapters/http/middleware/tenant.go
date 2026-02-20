package middleware

import (
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// TenantScope resolves the tenant ID and injects it into the request context.
func TenantScope(resolver ports.TenantResolver) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := resolver.Resolve(c.Request().Context())
			ctx := repository.WithTenantID(c.Request().Context(), tenantID)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
