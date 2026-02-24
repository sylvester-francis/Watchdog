package middleware

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
)

// requestMetadataKey is the context key for storing request metadata.
type requestMetadataKey struct{}

// RequestMetadata holds HTTP request details for tenant resolution.
type RequestMetadata struct {
	Host      string
	Headers   map[string]string
	RemoteIP  string
}

// RequestMetadataFromContext extracts request metadata from context.
func RequestMetadataFromContext(ctx context.Context) *RequestMetadata {
	if md, ok := ctx.Value(requestMetadataKey{}).(*RequestMetadata); ok {
		return md
	}
	return nil
}

// TenantScope resolves the tenant ID and injects it into the request context.
// For authenticated requests it first attempts to resolve the tenant from the
// user's database record via TenantID(). If that fails or returns empty, it
// falls back to the generic Resolve() which returns "default" in CE.
// Request metadata (host, headers) is injected into the context so EE's
// resolver can read X-Tenant-ID header and subdomain without interface changes.
func TenantScope(resolver ports.TenantResolver) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			// Inject request metadata into context for EE resolver.
			md := &RequestMetadata{
				Host:     req.Host,
				RemoteIP: c.RealIP(),
				Headers:  make(map[string]string),
			}
			for _, key := range []string{"X-Tenant-ID", "X-Forwarded-Host"} {
				if v := req.Header.Get(key); v != "" {
					md.Headers[key] = v
				}
			}
			ctx := context.WithValue(req.Context(), requestMetadataKey{}, md)

			var tenantID string

			// For authenticated requests, resolve from user's DB record.
			if userID, ok := GetUserID(c); ok {
				if tid, err := resolver.TenantID(ctx, userID); err == nil && tid != "" {
					tenantID = tid
				}
			}

			// Fall back to generic resolution (returns "default" in CE).
			if tenantID == "" {
				tenantID = resolver.Resolve(ctx)
			}

			ctx = repository.WithTenantID(ctx, tenantID)
			c.SetRequest(req.WithContext(ctx))
			return next(c)
		}
	}
}
