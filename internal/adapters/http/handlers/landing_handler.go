package handlers

import (
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// LandingHandler handles the public landing page and waitlist signups.
type LandingHandler struct {
	waitlistRepo ports.WaitlistRepository
	templates    *view.Templates
}

// NewLandingHandler creates a new LandingHandler.
func NewLandingHandler(waitlistRepo ports.WaitlistRepository, templates *view.Templates) *LandingHandler {
	return &LandingHandler{
		waitlistRepo: waitlistRepo,
		templates:    templates,
	}
}

// Page renders the landing page.
func (h *LandingHandler) Page(c echo.Context) error {
	return h.renderPage(c, "", "")
}

// JoinWaitlist handles the POST /waitlist form submission.
func (h *LandingHandler) JoinWaitlist(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))
	email = html.EscapeString(email)

	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return h.renderPage(c, "", "Please enter a valid email address.")
	}

	if len(email) > 255 {
		return h.renderPage(c, "", "Email address is too long.")
	}

	signup := &domain.WaitlistSignup{
		ID:        uuid.New(),
		Email:     email,
		CreatedAt: time.Now(),
	}

	if err := h.waitlistRepo.Create(c.Request().Context(), signup); err != nil {
		return h.renderPage(c, "", "Something went wrong. Please try again.")
	}

	return h.renderPage(c, "You're on the list! We'll be in touch soon.", "")
}

func (h *LandingHandler) renderPage(c echo.Context, success, errMsg string) error {
	return c.Render(http.StatusOK, "landing.html", map[string]interface{}{
		"Title":   "WatchDog - Monitor Services Behind Your Firewall",
		"Year":    time.Now().Year(),
		"Success": success,
		"Error":   errMsg,
	})
}
