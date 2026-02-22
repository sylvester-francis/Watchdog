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
