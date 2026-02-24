package handlers

import (
	"fmt"
	"strings"
	"time"
)

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func formatMetricReading(target, msg string) string {
	parts := strings.SplitN(target, ":", 2)
	if len(parts) < 1 {
		return ""
	}
	metric := strings.ToUpper(parts[0][:1]) + parts[0][1:]

	idx := strings.Index(msg, "usage ")
	if idx == -1 {
		return ""
	}
	rest := msg[idx+6:]
	pctIdx := strings.Index(rest, "%")
	if pctIdx == -1 {
		return ""
	}
	return metric + " " + rest[:pctIdx] + "%"
}
