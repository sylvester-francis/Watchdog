package view

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// Templates implements echo.Renderer for Go templates.
// Each page gets its own cloned template set so that {{define "content"}}
// blocks don't collide across pages (Go templates have a flat namespace).
type Templates struct {
	pages map[string]*template.Template
}

// NewTemplates creates a new Templates instance by loading all templates.
// For each page, it clones the base (layouts + partials) and parses the
// page file into the clone, giving each page its own "content" definition.
func NewTemplates(dir string) (*Templates, error) {
	funcMap := template.FuncMap{
		"formatTime":       formatTime,
		"formatDuration":   formatDuration,
		"formatTimeAgo":    formatTimeAgo,
		"formatTimeAgoPtr": formatTimeAgoPtr,
		"formatPercent":    formatPercent,
		"statusColor":      statusColor,
		"statusBgColor":    statusBgColor,
		"statusIcon":       statusIcon,
		"monitorTypeIcon":  monitorTypeIcon,
		"lower":            func(v interface{}) string { return strings.ToLower(fmt.Sprint(v)) },
		"upper":            func(v interface{}) string { return strings.ToUpper(fmt.Sprint(v)) },
		"title":            cases.Title(language.English).String,

		"add":              add,
		"sub":              sub,
		"dict":             dict,
		"toJSON":           toJSON,
		"deref":            deref,
	}

	// Build the shared base: layouts + partials
	base := template.New("").Funcs(funcMap)

	layoutPattern := filepath.Join(dir, "layouts", "*.html")
	base, err := base.ParseGlob(layoutPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse layouts: %w", err)
	}

	partialsPattern := filepath.Join(dir, "partials", "*.html")
	base, err = base.ParseGlob(partialsPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse partials: %w", err)
	}

	// For each page file, clone base and parse the page into the clone
	pages := make(map[string]*template.Template)

	pageFiles, err := filepath.Glob(filepath.Join(dir, "pages", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob pages: %w", err)
	}

	for _, pageFile := range pageFiles {
		// Clone the base template set
		clone, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone base for %s: %w", pageFile, err)
		}

		// Parse the page file into the clone
		pageContent, err := os.ReadFile(pageFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", pageFile, err)
		}

		_, err = clone.Parse(string(pageContent))
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", pageFile, err)
		}

		// Extract the page name (e.g. "dashboard.html" from the {{define "dashboard.html"}} block)
		// We register by filename so c.Render("dashboard.html", ...) finds the right set
		name := filepath.Base(pageFile)
		pages[name] = clone
	}

	return &Templates{pages: pages}, nil
}

// Render implements echo.Renderer.
func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Add common data to all templates
	viewData := map[string]interface{}{
		"Data": data,
	}

	// If data is already a map, merge it
	if m, ok := data.(map[string]interface{}); ok {
		for k, v := range m {
			viewData[k] = v
		}
	}

	// Inject CSRF token if available (set by CSRF middleware)
	if csrf := c.Get("csrf"); csrf != nil {
		viewData["CSRFToken"] = csrf
	}

	// Inject authenticated user context for sidebar (IsAdmin, Plan, Username)
	if u := c.Get("authenticated_user"); u != nil {
		if user, ok := u.(*domain.User); ok {
			if _, exists := viewData["IsAdmin"]; !exists {
				viewData["IsAdmin"] = user.IsAdmin
			}
			if _, exists := viewData["Plan"]; !exists {
				viewData["Plan"] = user.Plan.String()
			}
			if _, exists := viewData["Username"]; !exists {
				viewData["Username"] = user.Username
			}
		}
	}

	// Look up the page-specific template set
	tmpl, ok := t.pages[name]
	if !ok {
		// Fallback: search all page template sets for inline-defined templates
		// (e.g. "incident_row" defined inside incidents.html)
		for _, pt := range t.pages {
			if pt.Lookup(name) != nil {
				tmpl = pt
				ok = true
				break
			}
		}
	}
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}

	if err := tmpl.ExecuteTemplate(w, name, viewData); err != nil {
		return fmt.Errorf("template %q: %w", name, err)
	}
	return nil
}

// Template functions

// formatTime formats a time.Time to a readable string.
func formatTime(t time.Time) string {
	return t.Format("Jan 02, 2006 15:04:05")
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// formatTimeAgo formats a time as a relative duration (e.g., "5 minutes ago").
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// formatTimeAgoPtr formats a *time.Time as a relative duration.
func formatTimeAgoPtr(t *time.Time) string {
	if t == nil {
		return "never"
	}
	return formatTimeAgo(*t)
}

// formatPercent formats a float64 as a percentage string (e.g. "99.9").
func formatPercent(f float64) string {
	if f >= 100 {
		return "100"
	}
	return fmt.Sprintf("%.1f", f)
}

// statusColor returns a Tailwind text color class based on status.
func statusColor(status string) string {
	switch strings.ToLower(status) {
	case "online", "up", "resolved":
		return "text-green-400"
	case "offline", "down", "error":
		return "text-red-400"
	case "degraded", "timeout", "acknowledged":
		return "text-yellow-400"
	case "pending", "unknown", "open":
		return "text-gray-400"
	default:
		return "text-gray-400"
	}
}

// statusBgColor returns a Tailwind background color class based on status.
func statusBgColor(status string) string {
	switch strings.ToLower(status) {
	case "online", "up", "resolved":
		return "bg-green-500"
	case "offline", "down", "error":
		return "bg-red-500"
	case "degraded", "timeout", "acknowledged":
		return "bg-yellow-500"
	case "pending", "unknown", "open":
		return "bg-gray-500"
	default:
		return "bg-gray-500"
	}
}

// statusIcon returns an icon/emoji based on status.
func statusIcon(status string) string {
	switch strings.ToLower(status) {
	case "online", "up", "resolved":
		return "check_circle"
	case "offline", "down", "error":
		return "error"
	case "degraded", "timeout", "acknowledged":
		return "warning"
	case "pending", "unknown", "open":
		return "help"
	default:
		return "help"
	}
}

// monitorTypeIcon returns an icon name for monitor type.
func monitorTypeIcon(t domain.MonitorType) string {
	switch t {
	case domain.MonitorTypePing:
		return "network_ping"
	case domain.MonitorTypeHTTP:
		return "language"
	case domain.MonitorTypeTCP:
		return "cable"
	case domain.MonitorTypeDNS:
		return "dns"
	default:
		return "monitor_heart"
	}
}

// add adds two integers.
func add(a, b int) int {
	return a + b
}

// sub subtracts two integers.
func sub(a, b int) int {
	return a - b
}

// dict creates a map from key-value pairs.
func dict(values ...interface{}) map[string]interface{} {
	if len(values)%2 != 0 {
		return nil
	}
	result := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		result[key] = values[i+1]
	}
	return result
}

// deref dereferences a pointer value for use in templates.
// Supports *int, *string, and *time.Time.
func deref(v interface{}) interface{} {
	switch p := v.(type) {
	case *int:
		if p != nil {
			return *p
		}
	case *string:
		if p != nil {
			return *p
		}
	case *time.Time:
		if p != nil {
			return *p
		}
	}
	return nil
}

// toJSON serializes a value to JSON for embedding in templates.
// Used to pass Go data to Alpine.js/Chart.js on the client side.
func toJSON(v interface{}) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(b)
}
