package notify_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
)

func TestEmailNotifier_IncidentOpened(t *testing.T) {
	// Connecting to a non-existent SMTP server should return a NotifierError.
	notifier := notify.NewEmailNotifier(notify.EmailConfig{
		Host:     "127.0.0.1",
		Port:     1, // Nothing listening on port 1
		Username: "test@example.com",
		Password: "secret",
		From:     "watchdog@example.com",
		To:       "oncall@example.com",
	})

	incident := domain.NewIncident(uuid.New())
	monitor := domain.NewMonitor(uuid.New(), "Test HTTP Monitor", domain.MonitorTypeHTTP, "https://example.com")

	err := notifier.NotifyIncidentOpened(context.Background(), incident, monitor)

	require.Error(t, err)
	assert.True(t, notify.IsNotifierError(err), "expected NotifierError, got: %T", err)
	assert.Contains(t, err.Error(), "email")
}

func TestEmailNotifier_IncidentResolved(t *testing.T) {
	// Connecting to a non-existent SMTP server should return a NotifierError.
	notifier := notify.NewEmailNotifier(notify.EmailConfig{
		Host:     "127.0.0.1",
		Port:     1,
		Username: "test@example.com",
		Password: "secret",
		From:     "watchdog@example.com",
		To:       "oncall@example.com",
	})

	incident := domain.NewIncident(uuid.New())
	now := time.Now()
	incident.ResolvedAt = &now
	incident.Status = domain.IncidentStatusResolved

	monitor := domain.NewMonitor(uuid.New(), "Test DNS Monitor", domain.MonitorTypeDNS, "example.com")

	err := notifier.NotifyIncidentResolved(context.Background(), incident, monitor)

	require.Error(t, err)
	assert.True(t, notify.IsNotifierError(err), "expected NotifierError, got: %T", err)
	assert.Contains(t, err.Error(), "email")
	assert.Contains(t, err.Error(), "send mail")
}
