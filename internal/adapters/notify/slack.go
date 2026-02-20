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

// SlackNotifier sends notifications to a Slack webhook.
type SlackNotifier struct {
	webhookURL string
	httpClient *http.Client
}

// NewSlackNotifier creates a new Slack notifier.
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NotifyIncidentOpened sends a notification when an incident is opened.
func (s *SlackNotifier) NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	payload := slackPayload{
		Attachments: []slackAttachment{
			{
				Color:  "#FF0000",
				Title:  fmt.Sprintf("Incident Opened: %s", monitor.Name),
				Text:   fmt.Sprintf("Monitor *%s* is DOWN", monitor.Name),
				Fields: incidentFields(incident, monitor),
				Footer: "WatchDog Monitoring",
				Ts:     incident.StartedAt.Unix(),
			},
		},
	}

	return s.send(ctx, payload)
}

// NotifyIncidentResolved sends a notification when an incident is resolved.
func (s *SlackNotifier) NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	fields := incidentFields(incident, monitor)
	fields = append(fields, slackField{
		Title: "Duration",
		Value: formatDuration(incident.Duration()),
		Short: true,
	})

	var ts int64
	if incident.ResolvedAt != nil {
		ts = incident.ResolvedAt.Unix()
	}

	payload := slackPayload{
		Attachments: []slackAttachment{
			{
				Color:  "#00FF00",
				Title:  fmt.Sprintf("Incident Resolved: %s", monitor.Name),
				Text:   fmt.Sprintf("Monitor *%s* is UP", monitor.Name),
				Fields: fields,
				Footer: "WatchDog Monitoring",
				Ts:     ts,
			},
		},
	}

	return s.send(ctx, payload)
}

func (s *SlackNotifier) send(ctx context.Context, payload slackPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifierError{Notifier: "slack", Err: fmt.Errorf("marshal payload: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return &NotifierError{Notifier: "slack", Err: fmt.Errorf("create request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return &NotifierError{Notifier: "slack", Err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &NotifierError{Notifier: "slack", Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
	}

	return nil
}

func incidentFields(incident *domain.Incident, monitor *domain.Monitor) []slackField {
	return []slackField{
		{Title: "Monitor", Value: monitor.Name, Short: true},
		{Title: "Type", Value: string(monitor.Type), Short: true},
		{Title: "Target", Value: monitor.Target, Short: false},
		{Title: "Started At", Value: incident.StartedAt.Format(time.RFC3339), Short: true},
	}
}

type slackPayload struct {
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Color  string       `json:"color"`
	Title  string       `json:"title"`
	Text   string       `json:"text"`
	Fields []slackField `json:"fields,omitempty"`
	Footer string       `json:"footer,omitempty"`
	Ts     int64        `json:"ts,omitempty"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}
