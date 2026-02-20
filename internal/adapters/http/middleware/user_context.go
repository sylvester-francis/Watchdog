package middleware

import (
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// userContextKey is the context key for the authenticated user.
const userContextKey = "authenticated_user"

// UserContext creates middleware that loads the full user record and stores it
// in the echo context. Must be used after AuthRequired.
func UserContext(userRepo ports.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, ok := GetUserID(c)
			if !ok {
				return next(c)
			}

			user, err := userRepo.GetByID(c.Request().Context(), userID)
			if err != nil || user == nil {
				return next(c)
			}

			c.Set(userContextKey, user)
			return next(c)
		}
	}
}

// GetUser retrieves the authenticated user from the context.
func GetUser(c echo.Context) *domain.User {
	u := c.Get(userContextKey)
	if u == nil {
		return nil
	}
	user, ok := u.(*domain.User)
	if !ok {
		return nil
	}
	return user
}
