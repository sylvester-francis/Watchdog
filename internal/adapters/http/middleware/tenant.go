package middleware

import (
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
)

// TenantScope resolves the tenant ID and injects it into the request context.
// For authenticated requests it first attempts to resolve the tenant from the
// user's database record via TenantID(). If that fails or returns empty, it
// falls back to the generic Resolve() which returns "default" in CE.
func TenantScope(resolver ports.TenantResolver) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var tenantID string

			// For authenticated requests, resolve from user's DB record.
			if userID, ok := GetUserID(c); ok {
				if tid, err := resolver.TenantID(c.Request().Context(), userID); err == nil && tid != "" {
					tenantID = tid
				}
			}

			// Fall back to generic resolution (returns "default" in CE).
			if tenantID == "" {
				tenantID = resolver.Resolve(c.Request().Context())
			}

			ctx := repository.WithTenantID(c.Request().Context(), tenantID)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
