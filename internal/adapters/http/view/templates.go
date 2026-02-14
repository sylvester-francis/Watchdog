package view

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
)

// Templates implements echo.Renderer for Go templates.
type Templates struct {
	templates *template.Template
}

// NewTemplates creates a new Templates instance by loading all templates.
func NewTemplates(dir string) (*Templates, error) {
	funcMap := template.FuncMap{
		"formatTime":      formatTime,
		"formatDuration":  formatDuration,
		"formatTimeAgo":   formatTimeAgo,
		"formatTimeAgoPtr": formatTimeAgoPtr,
		"statusColor":     statusColor,
		"statusBgColor":   statusBgColor,
		"statusIcon":      statusIcon,
		"monitorTypeIcon": monitorTypeIcon,
		"lower":           strings.ToLower,
		"upper":           strings.ToUpper,
		"title":           strings.Title,
		"safeHTML":        safeHTML,
		"add":             add,
		"sub":             sub,
		"dict":            dict,
	}

	// Parse all templates
	templates := template.New("").Funcs(funcMap)

	// Parse layouts
	layoutPattern := filepath.Join(dir, "layouts", "*.html")
	templates, err := templates.ParseGlob(layoutPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse layouts: %w", err)
	}

	// Parse pages
	pagesPattern := filepath.Join(dir, "pages", "*.html")
	templates, err = templates.ParseGlob(pagesPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pages: %w", err)
	}

	// Parse partials
	partialsPattern := filepath.Join(dir, "partials", "*.html")
	templates, err = templates.ParseGlob(partialsPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse partials: %w", err)
	}

	return &Templates{templates: templates}, nil
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

	return t.templates.ExecuteTemplate(w, name, viewData)
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

// statusColor returns a Tailwind text color class based on status.
func statusColor(status string) string {
	switch strings.ToLower(status) {
	case "online", "up", "resolved":
		return "text-green-400"
	case "offline", "down", "error":
		return "text-red-400"
	case "degraded", "timeout", "acknowledged":
		return "text-yellow-400"
	case "unknown", "open":
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
	case "unknown", "open":
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
	case "unknown", "open":
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

// safeHTML marks a string as safe HTML (use with caution).
func safeHTML(s string) template.HTML {
	return template.HTML(s)
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
