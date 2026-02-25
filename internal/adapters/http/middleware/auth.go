package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/ports"
)

// UserIDKey is the context key for the authenticated user ID.
const UserIDKey = "user_id"

// SessionName is the name of the session cookie.
const SessionName = "session"

// sessionStoreKey is the context key for the session store.
const sessionStoreKey = "session_store"

// issuedAtKey is the session value key for when the session was created (H-002).
const issuedAtKey = "issued_at"

// maxSessionAge is the server-side session lifetime (H-002).
const maxSessionAge = 24 * time.Hour

// SessionMiddleware creates middleware that injects the session store into the context.
func SessionMiddleware(store *sessions.CookieStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(sessionStoreKey, store)
			return next(c)
		}
	}
}

// getSession retrieves the session from the request.
func getSession(c echo.Context) (*sessions.Session, error) {
	store := c.Get(sessionStoreKey)
	if store == nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "session store not configured")
	}
	return store.(*sessions.CookieStore).Get(c.Request(), SessionName)
}

// NoCacheHeaders is middleware that prevents browser caching of responses.
// This ensures that after logout, the browser back button triggers a fresh
// server request (which will redirect to /login) instead of showing cached pages.
func NoCacheHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		h := c.Response().Header()
		h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		h.Set("Pragma", "no-cache")
		h.Set("Expires", "0")
		return next(c)
	}
}

// AuthRequired is middleware that requires authentication.
// It redirects to /login if the user is not authenticated.
// If user_id is already set in context (e.g. by upstream Kratos middleware),
// authentication is considered complete and session check is skipped.
func AuthRequired(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Get(UserIDKey) != nil {
			return next(c)
		}

		sess, err := getSession(c)
		if err != nil {
			return c.Redirect(http.StatusFound, "/login")
		}

		userID := sess.Values[UserIDKey]
		if userID == nil {
			return c.Redirect(http.StatusFound, "/login")
		}

		// H-002: enforce server-side session age limit.
		if issuedAt, ok := sess.Values[issuedAtKey].(int64); ok {
			if time.Since(time.Unix(issuedAt, 0)) > maxSessionAge {
				_ = ClearSession(c)
				return c.Redirect(http.StatusFound, "/login")
			}
		}

		// Store user ID in context for handlers to use
		c.Set(UserIDKey, userID)

		return next(c)
	}
}

// AuthRequiredAPI is middleware for API endpoints that requires authentication.
// It returns 401 Unauthorized instead of redirecting.
// If user_id is already set in context (e.g. by upstream Kratos middleware),
// authentication is considered complete and session check is skipped.
func AuthRequiredAPI(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Get(UserIDKey) != nil {
			return next(c)
		}

		sess, err := getSession(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		userID := sess.Values[UserIDKey]
		if userID == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		// H-002: enforce server-side session age limit.
		if issuedAt, ok := sess.Values[issuedAtKey].(int64); ok {
			if time.Since(time.Unix(issuedAt, 0)) > maxSessionAge {
				_ = ClearSession(c)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "session expired",
				})
			}
		}

		// Store user ID in context for handlers to use
		c.Set(UserIDKey, userID)

		return next(c)
	}
}

// GetUserID retrieves the authenticated user's ID from the context.
func GetUserID(c echo.Context) (uuid.UUID, bool) {
	userID := c.Get(UserIDKey)
	if userID == nil {
		return uuid.Nil, false
	}

	// Try to parse as string first (from session)
	if idStr, ok := userID.(string); ok {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return uuid.Nil, false
		}
		return id, true
	}

	// Try to parse as UUID directly
	if id, ok := userID.(uuid.UUID); ok {
		return id, true
	}

	return uuid.Nil, false
}

// SetUserID stores the user ID and issuance timestamp in the session (H-002).
func SetUserID(c echo.Context, userID uuid.UUID) error {
	sess, err := getSession(c)
	if err != nil {
		return err
	}

	sess.Values[UserIDKey] = userID.String()
	sess.Values[issuedAtKey] = time.Now().Unix()
	return sess.Save(c.Request(), c.Response())
}

// ClearSession clears the user session (logout).
func ClearSession(c echo.Context) error {
	sess, err := getSession(c)
	if err != nil {
		return err
	}

	sess.Values = make(map[any]any)
	sess.Options.MaxAge = -1
	return sess.Save(c.Request(), c.Response())
}

// SessionPasswordCheck invalidates sessions issued before the user's last
// password change (H-003). Must run after AuthRequired sets UserIDKey.
func SessionPasswordCheck(userRepo ports.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := getSession(c)
			if err != nil {
				return next(c)
			}

			issuedAt, ok := sess.Values[issuedAtKey].(int64)
			if !ok {
				return next(c) // legacy session without issued_at
			}

			userID, ok := GetUserID(c)
			if !ok {
				return next(c)
			}

			user, err := userRepo.GetByID(c.Request().Context(), userID)
			if err != nil || user == nil {
				return next(c)
			}

			if user.PasswordChangedAt != nil && user.PasswordChangedAt.Unix() > issuedAt {
				_ = ClearSession(c)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "session invalidated — password was changed",
				})
			}

			return next(c)
		}
	}
}

// IsAuthenticated checks if the user has a valid session.
// This can be used without the AuthRequired middleware.
func IsAuthenticated(c echo.Context) bool {
	sess, err := getSession(c)
	if err != nil {
		return false
	}
	return sess.Values[UserIDKey] != nil
}
