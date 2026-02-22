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

const telegramDefaultBaseURL = "https://api.telegram.org"

// TelegramNotifier sends notifications via Telegram Bot API.
type TelegramNotifier struct {
	botToken   string
	chatID     string
	baseURL    string
	httpClient *http.Client
}

// NewTelegramNotifier creates a new Telegram notifier.
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		baseURL:  telegramDefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetBaseURL overrides the Telegram API base URL (useful for testing).
func (t *TelegramNotifier) SetBaseURL(url string) {
	t.baseURL = url
}

// SetHTTPClient overrides the HTTP client (useful for testing).
func (t *TelegramNotifier) SetHTTPClient(client *http.Client) {
	t.httpClient = client
}

// NotifyIncidentOpened sends a Telegram message when an incident is opened.
func (t *TelegramNotifier) NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	text := fmt.Sprintf(
		"🔴 *Incident Opened*\n\n*Monitor:* %s\n*Type:* %s\n*Target:* `%s`\n*Started:* %s",
		escapeMarkdown(monitor.Name),
		string(monitor.Type),
		monitor.Target,
		incident.StartedAt.Format(time.RFC3339),
	)

	return t.send(ctx, text)
}

// NotifyIncidentResolved sends a Telegram message when an incident is resolved.
func (t *TelegramNotifier) NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	text := fmt.Sprintf(
		"🟢 *Incident Resolved*\n\n*Monitor:* %s\n*Type:* %s\n*Target:* `%s`\n*Duration:* %s",
		escapeMarkdown(monitor.Name),
		string(monitor.Type),
		monitor.Target,
		formatDuration(incident.Duration()),
	)

	return t.send(ctx, text)
}

func (t *TelegramNotifier) send(ctx context.Context, text string) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", t.baseURL, t.botToken)

	payload := map[string]string{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifierError{Notifier: "telegram", Err: fmt.Errorf("marshal payload: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return &NotifierError{Notifier: "telegram", Err: fmt.Errorf("create request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return &NotifierError{Notifier: "telegram", Err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &NotifierError{Notifier: "telegram", Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
	}

	return nil
}

// escapeMarkdown escapes special Markdown characters for Telegram.
func escapeMarkdown(s string) string {
	replacer := []string{"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "`", "\\`"}
	result := s
	for i := 0; i < len(replacer); i += 2 {
		// Simple escape — replace each char
		for j := range result {
			if string(result[j]) == replacer[i] {
				result = result[:j] + replacer[i+1] + result[j+1:]
				break
			}
		}
	}
	return result
}
