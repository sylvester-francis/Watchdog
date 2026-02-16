package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// APITokenAuth creates middleware that authenticates requests via Bearer token.
// It looks up the token hash in the database, validates expiry, and sets the
// user ID in the context so downstream handlers can use GetUserID().
func APITokenAuth(tokenRepo ports.APITokenRepository, userRepo ports.UserRepository) echo.MiddlewareFunc {
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

			// Verify user still exists
			user, err := userRepo.GetByID(c.Request().Context(), token.UserID)
			if err != nil || user == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "token owner not found",
				})
			}

			// Set user ID in context (same key as session auth)
			c.Set(UserIDKey, token.UserID.String())

			// Update last used (fire-and-forget)
			go func() {
				_ = tokenRepo.UpdateLastUsed(c.Request().Context(), token.ID)
			}()

			return next(c)
		}
	}
}
