package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StatusPage represents a public status page showing monitor health.
type StatusPage struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Slug        string
	Description string
	IsPublic    bool
	MonitorIDs  []uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

// GenerateSlug creates a URL-safe slug from a name.
func GenerateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "status"
	}
	return slug
}

// NewStatusPage creates a new status page.
func NewStatusPage(userID uuid.UUID, name, slug string) *StatusPage {
	return &StatusPage{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		Slug:      slug,
		IsPublic:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
