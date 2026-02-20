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

// WebhookNotifier sends notifications to a generic webhook URL.
type WebhookNotifier struct {
	url        string
	httpClient *http.Client
}

// NewWebhookNotifier creates a new generic webhook notifier.
func NewWebhookNotifier(url string) *WebhookNotifier {
	return &WebhookNotifier{
		url: url,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NotifyIncidentOpened sends a notification when an incident is opened.
func (w *WebhookNotifier) NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	payload := webhookPayload{
		Event:     "incident.opened",
		Timestamp: incident.StartedAt,
		Incident: webhookIncident{
			ID:        incident.ID.String(),
			MonitorID: incident.MonitorID.String(),
			Status:    string(incident.Status),
			StartedAt: incident.StartedAt,
		},
		Monitor: webhookMonitor{
			ID:     monitor.ID.String(),
			Name:   monitor.Name,
			Type:   string(monitor.Type),
			Target: monitor.Target,
		},
	}

	return w.send(ctx, payload)
}

// NotifyIncidentResolved sends a notification when an incident is resolved.
func (w *WebhookNotifier) NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	payload := webhookPayload{
		Event:     "incident.resolved",
		Timestamp: time.Now(),
		Incident: webhookIncident{
			ID:         incident.ID.String(),
			MonitorID:  incident.MonitorID.String(),
			Status:     string(incident.Status),
			StartedAt:  incident.StartedAt,
			ResolvedAt: incident.ResolvedAt,
		},
		Monitor: webhookMonitor{
			ID:     monitor.ID.String(),
			Name:   monitor.Name,
			Type:   string(monitor.Type),
			Target: monitor.Target,
		},
	}

	return w.send(ctx, payload)
}

func (w *WebhookNotifier) send(ctx context.Context, payload webhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifierError{Notifier: "webhook", Err: fmt.Errorf("marshal payload: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return &NotifierError{Notifier: "webhook", Err: fmt.Errorf("create request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return &NotifierError{Notifier: "webhook", Err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &NotifierError{Notifier: "webhook", Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
	}

	return nil
}

type webhookPayload struct {
	Event     string          `json:"event"`
	Timestamp time.Time       `json:"timestamp"`
	Incident  webhookIncident `json:"incident"`
	Monitor   webhookMonitor  `json:"monitor"`
}

type webhookIncident struct {
	ID         string     `json:"id"`
	MonitorID  string     `json:"monitor_id"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type webhookMonitor struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Target string `json:"target"`
}
