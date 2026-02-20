package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// Discord embed colors.
const (
	colorRed   = 0xFF0000 // Incident opened
	colorGreen = 0x00FF00 // Incident resolved
)

// DiscordNotifier sends notifications to a Discord webhook.
type DiscordNotifier struct {
	webhookURL string
	httpClient *http.Client
}

// NewDiscordNotifier creates a new Discord notifier.
func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NotifyIncidentOpened sends a notification when an incident is opened.
func (d *DiscordNotifier) NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	embed := discordEmbed{
		Title:       fmt.Sprintf("🚨 Incident Opened: %s", monitor.Name),
		Description: fmt.Sprintf("Monitor **%s** is DOWN", monitor.Name),
		Color:       colorRed,
		Fields: []discordField{
			{Name: "Monitor", Value: monitor.Name, Inline: true},
			{Name: "Type", Value: string(monitor.Type), Inline: true},
			{Name: "Target", Value: monitor.Target, Inline: false},
			{Name: "Started At", Value: incident.StartedAt.Format(time.RFC3339), Inline: true},
		},
		Timestamp: incident.StartedAt.Format(time.RFC3339),
		Footer: discordFooter{
			Text: "WatchDog Monitoring",
		},
	}

	return d.sendWebhook(ctx, embed)
}

// NotifyIncidentResolved sends a notification when an incident is resolved.
func (d *DiscordNotifier) NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	duration := incident.Duration()
	durationStr := formatDuration(duration)

	embed := discordEmbed{
		Title:       fmt.Sprintf("✅ Incident Resolved: %s", monitor.Name),
		Description: fmt.Sprintf("Monitor **%s** is UP", monitor.Name),
		Color:       colorGreen,
		Fields: []discordField{
			{Name: "Monitor", Value: monitor.Name, Inline: true},
			{Name: "Type", Value: string(monitor.Type), Inline: true},
			{Name: "Target", Value: monitor.Target, Inline: false},
			{Name: "Started At", Value: incident.StartedAt.Format(time.RFC3339), Inline: true},
			{Name: "Duration", Value: durationStr, Inline: true},
		},
		Footer: discordFooter{
			Text: "WatchDog Monitoring",
		},
	}

	if incident.ResolvedAt != nil {
		embed.Timestamp = incident.ResolvedAt.Format(time.RFC3339)
	}

	return d.sendWebhook(ctx, embed)
}

// sendWebhook sends a webhook message to Discord.
func (d *DiscordNotifier) sendWebhook(ctx context.Context, embed discordEmbed) error {
	payload := discordWebhookPayload{
		Embeds: []discordEmbed{embed},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifierError{Notifier: "discord", Err: fmt.Errorf("marshal payload: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return &NotifierError{Notifier: "discord", Err: fmt.Errorf("create request: %w", err)}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return &NotifierError{Notifier: "discord", Err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &NotifierError{Notifier: "discord", Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
	}

	return nil
}

// Discord webhook payload structures.
type discordWebhookPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

type discordEmbed struct {
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color,omitempty"`
	Fields      []discordField `json:"fields,omitempty"`
	Footer      discordFooter  `json:"footer,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type discordFooter struct {
	Text string `json:"text,omitempty"`
}

// formatDuration formats a duration in a human-readable format.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
