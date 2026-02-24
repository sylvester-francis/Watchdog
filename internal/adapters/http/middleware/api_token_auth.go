package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// HybridAuth creates middleware that authenticates requests via Bearer token first,
// falling back to session cookie auth. This allows both the public API (token-based)
// and the SvelteKit SPA (cookie-based) to use the same /api/v1/* endpoints.
// Returns JSON 401 on failure (never redirects).
func HybridAuth(tokenRepo ports.APITokenRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Try Bearer token first
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				plaintext := strings.TrimPrefix(authHeader, "Bearer ")
				if strings.HasPrefix(plaintext, "wd_") {
					hash := domain.HashToken(plaintext)
					token, err := tokenRepo.GetByTokenHash(c.Request().Context(), hash)
					if err == nil && token != nil && !token.IsExpired() {
						c.Set(UserIDKey, token.UserID.String())
						c.Set("token_scope", string(token.Scope))

						ip := c.RealIP()
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
							defer cancel()
							_ = tokenRepo.UpdateLastUsed(ctx, token.ID, ip)
						}()

						return next(c)
					}
				}
				// Invalid Bearer token — fall through to session auth
			}

			// 2. Fall back to session cookie
			sess, err := getSession(c)
			if err == nil {
				userID := sess.Values[UserIDKey]
				if userID != nil {
					c.Set(UserIDKey, userID)
					return next(c)
				}
			}

			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}
	}
}

// RequireWriteScope creates middleware that rejects mutating requests (POST, PUT, DELETE)
// from API tokens with read_only scope. Session-based users (no token_scope set) are
// always allowed through because they authenticated via the web UI.
func RequireWriteScope() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			scope, _ := c.Get("token_scope").(string)
			// No scope set means session-based auth — allow all methods
			if scope == "" {
				return next(c)
			}

			method := c.Request().Method
			if scope == string(domain.TokenScopeReadOnly) &&
				(method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete) {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "read_only token cannot perform write operations",
				})
			}

			return next(c)
		}
	}
}

// APITokenAuth creates middleware that authenticates requests via Bearer token.
// It looks up the token hash in the database, validates expiry, and sets the
// user ID in the context so downstream handlers can use GetUserID().
// The user existence check is omitted — if the token exists and isn't expired,
// the user must exist (enforced by ON DELETE CASCADE). TenantScope middleware
// (which runs after this) resolves the correct tenant from the user ID.
func APITokenAuth(tokenRepo ports.APITokenRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "missing or invalid Authorization header",
				})
			}

			plaintext := strings.TrimPrefix(auth, "Bearer ")
			if !strings.HasPrefix(plaintext, "wd_") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid token format",
				})
			}

			hash := domain.HashToken(plaintext)
			token, err := tokenRepo.GetByTokenHash(c.Request().Context(), hash)
			if err != nil || token == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid token",
				})
			}

			if token.IsExpired() {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "token expired",
				})
			}

			// Set user ID and token scope in context (same key as session auth)
			c.Set(UserIDKey, token.UserID.String())
			c.Set("token_scope", string(token.Scope))

			// Update last used + IP (fire-and-forget with dedicated context)
			ip := c.RealIP()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = tokenRepo.UpdateLastUsed(ctx, token.ID, ip)
			}()

			return next(c)
		}
	}
}
