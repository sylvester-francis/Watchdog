package notify

import (
	"context"
	"errors"
	"fmt"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// Notifier defines the interface for sending alert notifications.
type Notifier interface {
	NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
	NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
	NotifyAgentOffline(ctx context.Context, agent *domain.Agent, affectedMonitors int) error
	NotifyAgentOnline(ctx context.Context, agent *domain.Agent, resolvedIncidents int) error
	NotifyAgentMaintenance(ctx context.Context, agent *domain.Agent, windowName string) error
}

// MultiNotifier sends notifications to multiple notifiers.
// It collects errors from all notifiers and returns them as a combined error.
type MultiNotifier struct {
	notifiers []Notifier
}

// NewMultiNotifier creates a new MultiNotifier with the given notifiers.
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{
		notifiers: notifiers,
	}
}

// AddNotifier adds a notifier to the list.
func (m *MultiNotifier) AddNotifier(n Notifier) {
	m.notifiers = append(m.notifiers, n)
}

// NotifyIncidentOpened sends incident opened notifications to all notifiers.
func (m *MultiNotifier) NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	var errs []error

	for _, n := range m.notifiers {
		if err := n.NotifyIncidentOpened(ctx, incident, monitor); err != nil {
			errs = append(errs, err)
		}
	}

	return combineErrors(errs)
}

// NotifyIncidentResolved sends incident resolved notifications to all notifiers.
func (m *MultiNotifier) NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	var errs []error

	for _, n := range m.notifiers {
		if err := n.NotifyIncidentResolved(ctx, incident, monitor); err != nil {
			errs = append(errs, err)
		}
	}

	return combineErrors(errs)
}

// NotifyAgentOffline sends agent offline notifications to all notifiers.
func (m *MultiNotifier) NotifyAgentOffline(ctx context.Context, agent *domain.Agent, affectedMonitors int) error {
	var errs []error
	for _, n := range m.notifiers {
		if err := n.NotifyAgentOffline(ctx, agent, affectedMonitors); err != nil {
			errs = append(errs, err)
		}
	}
	return combineErrors(errs)
}

// NotifyAgentOnline sends agent online notifications to all notifiers.
func (m *MultiNotifier) NotifyAgentOnline(ctx context.Context, agent *domain.Agent, resolvedIncidents int) error {
	var errs []error
	for _, n := range m.notifiers {
		if err := n.NotifyAgentOnline(ctx, agent, resolvedIncidents); err != nil {
			errs = append(errs, err)
		}
	}
	return combineErrors(errs)
}

// NotifyAgentMaintenance sends agent maintenance notifications to all notifiers.
func (m *MultiNotifier) NotifyAgentMaintenance(ctx context.Context, agent *domain.Agent, windowName string) error {
	var errs []error
	for _, n := range m.notifiers {
		if err := n.NotifyAgentMaintenance(ctx, agent, windowName); err != nil {
			errs = append(errs, err)
		}
	}
	return combineErrors(errs)
}

// NoOpNotifier is a notifier that does nothing.
// Useful as a default or for testing.
type NoOpNotifier struct{}

// NewNoOpNotifier creates a new no-op notifier.
func NewNoOpNotifier() *NoOpNotifier {
	return &NoOpNotifier{}
}

// NotifyIncidentOpened does nothing.
func (n *NoOpNotifier) NotifyIncidentOpened(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
	return nil
}

// NotifyIncidentResolved does nothing.
func (n *NoOpNotifier) NotifyIncidentResolved(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
	return nil
}

// NotifyAgentOffline does nothing.
func (n *NoOpNotifier) NotifyAgentOffline(_ context.Context, _ *domain.Agent, _ int) error {
	return nil
}

// NotifyAgentOnline does nothing.
func (n *NoOpNotifier) NotifyAgentOnline(_ context.Context, _ *domain.Agent, _ int) error {
	return nil
}

// NotifyAgentMaintenance does nothing.
func (n *NoOpNotifier) NotifyAgentMaintenance(_ context.Context, _ *domain.Agent, _ string) error {
	return nil
}

// combineErrors combines multiple errors into a single error.
func combineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	var combined error
	for _, err := range errs {
		if combined == nil {
			combined = err
		} else {
			combined = fmt.Errorf("%w; %v", combined, err)
		}
	}
	return combined
}

// formatInterval returns a human-readable check interval string.
func formatInterval(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("Every %ds", seconds)
	}
	if seconds < 3600 {
		m := seconds / 60
		if seconds%60 == 0 {
			return fmt.Sprintf("Every %dm", m)
		}
		return fmt.Sprintf("Every %dm %ds", m, seconds%60)
	}
	h := seconds / 3600
	return fmt.Sprintf("Every %dh", h)
}

// IsNotifierError checks if an error is a notifier-related error.
func IsNotifierError(err error) bool {
	var notifierErr *NotifierError
	return errors.As(err, &notifierErr)
}

// NotifierError represents an error from a specific notifier.
type NotifierError struct {
	Notifier string
	Err      error
}

func (e *NotifierError) Error() string {
	return fmt.Sprintf("%s: %v", e.Notifier, e.Err)
}

func (e *NotifierError) Unwrap() error {
	return e.Err
}
