package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
)

const pagerdutyDefaultEventsURL = "https://events.pagerduty.com/v2/enqueue"

// PagerDutyNotifier sends notifications via PagerDuty Events API v2.
type PagerDutyNotifier struct {
	routingKey string
	eventsURL  string
	httpClient *http.Client
}

// NewPagerDutyNotifier creates a new PagerDuty notifier.
func NewPagerDutyNotifier(routingKey string) *PagerDutyNotifier {
	return &PagerDutyNotifier{
		routingKey: routingKey,
		eventsURL:  pagerdutyDefaultEventsURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetEventsURL overrides the PagerDuty events URL (useful for testing).
func (p *PagerDutyNotifier) SetEventsURL(url string) {
	p.eventsURL = url
}

// SetHTTPClient overrides the HTTP client (useful for testing).
func (p *PagerDutyNotifier) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// NotifyIncidentOpened sends a trigger event to PagerDuty.
func (p *PagerDutyNotifier) NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	payload := pagerdutyEvent{
		RoutingKey:  p.routingKey,
		EventAction: "trigger",
		DedupKey:    incident.ID.String(),
		Payload: pagerdutyPayload{
			Summary:   fmt.Sprintf("Monitor %s is DOWN (%s)", monitor.Name, monitor.Target),
			Source:    "watchdog",
			Severity:  "critical",
			Timestamp: incident.StartedAt.Format(time.RFC3339),
			CustomDetails: map[string]string{
				"monitor_name": monitor.Name,
				"monitor_type": string(monitor.Type),
				"target":       monitor.Target,
			},
		},
	}

	return p.send(ctx, payload)
}

// NotifyIncidentResolved sends a resolve event to PagerDuty.
func (p *PagerDutyNotifier) NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	payload := pagerdutyEvent{
		RoutingKey:  p.routingKey,
		EventAction: "resolve",
		DedupKey:    incident.ID.String(),
		Payload: pagerdutyPayload{
			Summary:   fmt.Sprintf("Monitor %s is UP (%s)", monitor.Name, monitor.Target),
			Source:    "watchdog",
			Severity:  "info",
			Timestamp: time.Now().Format(time.RFC3339),
			CustomDetails: map[string]string{
				"monitor_name": monitor.Name,
				"duration":     formatDuration(incident.Duration()),
			},
		},
	}

	return p.send(ctx, payload)
}

func (p *PagerDutyNotifier) send(ctx context.Context, event pagerdutyEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return &NotifierError{Notifier: "pagerduty", Err: fmt.Errorf("marshal event: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.eventsURL, bytes.NewReader(body))
	if err != nil {
		return &NotifierError{Notifier: "pagerduty", Err: fmt.Errorf("create request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &NotifierError{Notifier: "pagerduty", Err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &NotifierError{Notifier: "pagerduty", Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
	}

	return nil
}

type pagerdutyEvent struct {
	RoutingKey  string           `json:"routing_key"`
	EventAction string           `json:"event_action"`
	DedupKey    string           `json:"dedup_key"`
	Payload     pagerdutyPayload `json:"payload"`
}

type pagerdutyPayload struct {
	Summary       string            `json:"summary"`
	Source        string            `json:"source"`
	Severity      string            `json:"severity"`
	Timestamp     string            `json:"timestamp"`
	CustomDetails map[string]string `json:"custom_details,omitempty"`
}
