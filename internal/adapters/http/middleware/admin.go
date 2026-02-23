package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/ports"
)

// AdminRequired creates middleware that restricts access to admin users only.
// Must be used after AuthRequired middleware. Redirects non-admins to /dashboard.
func AdminRequired(userRepo ports.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, ok := GetUserID(c)
			if !ok {
				return c.Redirect(http.StatusFound, "/dashboard")
			}

			user, err := userRepo.GetByID(c.Request().Context(), userID)
			if err != nil || user == nil || !user.IsAdmin {
				return c.Redirect(http.StatusFound, "/dashboard")
			}

			return next(c)
		}
	}
}

// AdminRequiredJSON creates middleware that restricts API access to admin users only.
// Returns JSON 403 for non-admins instead of redirecting.
func AdminRequiredJSON(userRepo ports.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, ok := GetUserID(c)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			user, err := userRepo.GetByID(c.Request().Context(), userID)
			if err != nil || user == nil || !user.IsAdmin {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "admin access required"})
			}

			return next(c)
		}
	}
}
