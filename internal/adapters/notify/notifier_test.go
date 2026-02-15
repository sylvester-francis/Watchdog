package notify_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
)

func testIncident() *domain.Incident {
	return domain.NewIncident(uuid.New())
}

func testMonitor() *domain.Monitor {
	return domain.NewMonitor(uuid.New(), "Test Monitor", domain.MonitorTypeHTTP, "https://example.com")
}

func TestNoOpNotifier_ReturnsNil(t *testing.T) {
	n := notify.NewNoOpNotifier()
	ctx := context.Background()

	err := n.NotifyIncidentOpened(ctx, testIncident(), testMonitor())
	assert.NoError(t, err)

	err = n.NotifyIncidentResolved(ctx, testIncident(), testMonitor())
	assert.NoError(t, err)
}

func TestMultiNotifier_AllSucceed(t *testing.T) {
	called := [2]bool{}

	n1 := &stubNotifier{
		openFn: func() error { called[0] = true; return nil },
	}
	n2 := &stubNotifier{
		openFn: func() error { called[1] = true; return nil },
	}

	multi := notify.NewMultiNotifier(n1, n2)
	err := multi.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	require.NoError(t, err)
	assert.True(t, called[0])
	assert.True(t, called[1])
}

func TestMultiNotifier_OneFails(t *testing.T) {
	expectedErr := errors.New("slack down")

	n1 := &stubNotifier{
		openFn: func() error { return nil },
	}
	n2 := &stubNotifier{
		openFn: func() error { return expectedErr },
	}

	multi := notify.NewMultiNotifier(n1, n2)
	err := multi.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slack down")
}

func TestMultiNotifier_AllFail(t *testing.T) {
	n1 := &stubNotifier{
		openFn: func() error { return errors.New("discord down") },
	}
	n2 := &stubNotifier{
		openFn: func() error { return errors.New("slack down") },
	}

	multi := notify.NewMultiNotifier(n1, n2)
	err := multi.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "discord down")
	assert.Contains(t, err.Error(), "slack down")
}

func TestMultiNotifier_Resolved(t *testing.T) {
	called := false
	n := &stubNotifier{
		resolveFn: func() error { called = true; return nil },
	}

	multi := notify.NewMultiNotifier(n)
	err := multi.NotifyIncidentResolved(context.Background(), testIncident(), testMonitor())

	require.NoError(t, err)
	assert.True(t, called)
}

func TestMultiNotifier_Resolved_OneFails(t *testing.T) {
	// Kills notifier.go:53 mutant — negated error check in resolved notification loop
	expectedErr := errors.New("webhook timeout")
	n := &stubNotifier{
		resolveFn: func() error { return expectedErr },
	}

	multi := notify.NewMultiNotifier(n)
	err := multi.NotifyIncidentResolved(context.Background(), testIncident(), testMonitor())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook timeout")
}

func TestNotifierError_Format(t *testing.T) {
	err := &notify.NotifierError{
		Notifier: "slack",
		Err:      errors.New("connection refused"),
	}

	assert.Equal(t, "slack: connection refused", err.Error())
}

func TestIsNotifierError(t *testing.T) {
	err := &notify.NotifierError{
		Notifier: "webhook",
		Err:      errors.New("timeout"),
	}

	assert.True(t, notify.IsNotifierError(err))
	assert.False(t, notify.IsNotifierError(errors.New("regular error")))
}

func TestIsNotifierError_Wrapped(t *testing.T) {
	inner := &notify.NotifierError{
		Notifier: "webhook",
		Err:      errors.New("timeout"),
	}
	wrapped := errors.Join(errors.New("outer"), inner)

	assert.True(t, notify.IsNotifierError(wrapped))
}

// stubNotifier is a test helper for creating simple inline notifiers.
type stubNotifier struct {
	openFn    func() error
	resolveFn func() error
}

func (s *stubNotifier) NotifyIncidentOpened(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
	if s.openFn != nil {
		return s.openFn()
	}
	return nil
}

func (s *stubNotifier) NotifyIncidentResolved(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
	if s.resolveFn != nil {
		return s.resolveFn()
	}
	return nil
}
