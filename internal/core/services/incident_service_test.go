package services_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/services"
	"github.com/sylvester-francis/watchdog/internal/testutil/mocks"
)

// testLogger returns a logger that writes to a buffer for assertion.
func testLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(handler), buf
}

func newTestIncidentService(
	incidentRepo *mocks.MockIncidentRepository,
	monitorRepo *mocks.MockMonitorRepository,
	notifier *mocks.MockNotifier,
	transactor *mocks.MockTransactor,
) *services.IncidentService {
	return services.NewIncidentService(incidentRepo, monitorRepo, &mocks.MockAgentRepository{}, &mocks.MockAlertChannelRepository{}, notifier, transactor, slog.Default())
}

// --- NewIncidentService nil logger ---

func TestNewIncidentService_NilLogger(t *testing.T) {
	// Mutant survivor: incident_service.go:31 — if logger == nil
	// If the nil guard is negated, logger stays nil and will PANIC on first log call.
	// We must trigger a code path that calls s.logger.Warn or s.logger.Error.
	incidentID := uuid.New()
	monitorID := uuid.New()

	callCount := 0
	incidentRepo := &mocks.MockIncidentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			callCount++
			if callCount == 1 {
				return &domain.Incident{ID: incidentID, MonitorID: monitorID, Status: domain.IncidentStatusOpen}, nil
			}
			// Second call (refresh) fails — triggers s.logger.Warn at line 116
			return nil, errors.New("refresh fail")
		},
		ResolveFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error { return nil },
	}

	svc := services.NewIncidentService(incidentRepo, monitorRepo, &mocks.MockAgentRepository{}, &mocks.MockAlertChannelRepository{}, &mocks.MockNotifier{}, &mocks.MockTransactor{}, nil)
	require.NotNil(t, svc)

	// This triggers logger.Warn — if nil guard was mutated, this panics
	err := svc.ResolveIncident(context.Background(), incidentID)
	require.NoError(t, err)
}

// --- ResolveIncident ---

func TestResolveIncident_Success(t *testing.T) {
	incidentID := uuid.New()
	monitorID := uuid.New()
	incident := &domain.Incident{
		ID:        incidentID,
		MonitorID: monitorID,
		Status:    domain.IncidentStatusOpen,
	}
	monitor := &domain.Monitor{
		ID:   monitorID,
		Name: "Test Monitor",
	}

	resolveCalled := false
	statusUpdated := false
	notified := false

	callCount := 0
	incidentRepo := &mocks.MockIncidentRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Incident, error) {
			callCount++
			assert.Equal(t, incidentID, id)
			return incident, nil
		},
		ResolveFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, incidentID, id)
			resolveCalled = true
			return nil
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Monitor, error) {
			return monitor, nil
		},
		UpdateStatusFn: func(_ context.Context, id uuid.UUID, status domain.MonitorStatus) error {
			assert.Equal(t, monitorID, id)
			assert.Equal(t, domain.MonitorStatusUp, status)
			statusUpdated = true
			return nil
		},
	}
	notifier := &mocks.MockNotifier{
		NotifyIncidentResolvedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
			notified = true
			return nil
		},
	}

	svc := newTestIncidentService(incidentRepo, monitorRepo, notifier, &mocks.MockTransactor{})

	err := svc.ResolveIncident(context.Background(), incidentID)

	require.NoError(t, err)
	assert.True(t, resolveCalled)
	assert.True(t, statusUpdated)
	assert.True(t, notified)
}

func TestResolveIncident_NotFound(t *testing.T) {
	incidentRepo := &mocks.MockIncidentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil
		},
	}

	svc := newTestIncidentService(incidentRepo, &mocks.MockMonitorRepository{}, &mocks.MockNotifier{}, &mocks.MockTransactor{})

	err := svc.ResolveIncident(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incident not found")
}

func TestResolveIncident_TransactionFails(t *testing.T) {
	incidentID := uuid.New()
	monitorID := uuid.New()
	txErr := errors.New("tx failed")

	incidentRepo := &mocks.MockIncidentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return &domain.Incident{ID: incidentID, MonitorID: monitorID, Status: domain.IncidentStatusOpen}, nil
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID}, nil
		},
	}
	notifier := &mocks.MockNotifier{
		NotifyIncidentResolvedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
			t.Fatal("notification should not be sent when transaction fails")
			return nil
		},
	}
	transactor := &mocks.MockTransactor{
		WithTransactionFn: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	}

	svc := newTestIncidentService(incidentRepo, monitorRepo, notifier, transactor)

	err := svc.ResolveIncident(context.Background(), incidentID)

	assert.ErrorIs(t, err, txErr)
}

func TestResolveIncident_NotificationFails_StillSucceeds(t *testing.T) {
	incidentID := uuid.New()
	monitorID := uuid.New()
	incident := &domain.Incident{ID: incidentID, MonitorID: monitorID, Status: domain.IncidentStatusOpen}

	incidentRepo := &mocks.MockIncidentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return incident, nil
		},
		ResolveFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error { return nil },
	}
	notifier := &mocks.MockNotifier{
		NotifyIncidentResolvedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
			return errors.New("slack down")
		},
	}

	svc := newTestIncidentService(incidentRepo, monitorRepo, notifier, &mocks.MockTransactor{})

	err := svc.ResolveIncident(context.Background(), incidentID)

	// Should succeed despite notification failure
	assert.NoError(t, err)
}

func TestResolveIncident_RefreshFails_LogsWarning(t *testing.T) {
	// Mutant survivor: incident_service.go:115 — if err != nil (refresh after resolve)
	// When refresh GetByID fails, a warning MUST be logged. The mutant skips the log.
	incidentID := uuid.New()
	monitorID := uuid.New()
	logger, logBuf := testLogger()

	getByIDCallCount := 0
	incidentRepo := &mocks.MockIncidentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			getByIDCallCount++
			if getByIDCallCount == 1 {
				return &domain.Incident{ID: incidentID, MonitorID: monitorID, Status: domain.IncidentStatusOpen}, nil
			}
			// Second call: refresh fails
			return nil, errors.New("db gone")
		},
		ResolveFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error { return nil },
	}
	notified := false
	notifier := &mocks.MockNotifier{
		NotifyIncidentResolvedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
			notified = true
			return nil
		},
	}

	svc := services.NewIncidentService(incidentRepo, monitorRepo, &mocks.MockAgentRepository{}, &mocks.MockAlertChannelRepository{}, notifier, &mocks.MockTransactor{}, logger)

	err := svc.ResolveIncident(context.Background(), incidentID)
	require.NoError(t, err)
	assert.Equal(t, 2, getByIDCallCount, "GetByID should be called twice (pre-resolve + refresh)")
	assert.False(t, notified, "notification should be skipped when refresh returns nil incident")
	assert.Contains(t, logBuf.String(), "failed to refresh incident", "warning must be logged when refresh fails")
}

func TestResolveIncident_RefreshReturnsNilIncident_SkipsNotification(t *testing.T) {
	// Mutant survivor: incident_service.go:120:13 — guard `monitor != nil && incident != nil`
	// When refresh returns nil for incident, notification must be skipped.
	incidentID := uuid.New()
	monitorID := uuid.New()

	callCount := 0
	incidentRepo := &mocks.MockIncidentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			callCount++
			if callCount == 1 {
				return &domain.Incident{ID: incidentID, MonitorID: monitorID, Status: domain.IncidentStatusOpen}, nil
			}
			return nil, nil // refresh returns nil
		},
		ResolveFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error { return nil },
	}
	notified := false
	notifier := &mocks.MockNotifier{
		NotifyIncidentResolvedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
			notified = true
			return nil
		},
	}

	svc := newTestIncidentService(incidentRepo, monitorRepo, notifier, &mocks.MockTransactor{})

	err := svc.ResolveIncident(context.Background(), incidentID)
	require.NoError(t, err)
	assert.False(t, notified, "should skip notification when incident is nil after refresh")
}

func TestResolveIncident_NotificationFails_LogsError(t *testing.T) {
	// Mutant survivor: incident_service.go:121:88 — notifyErr != nil negated
	// When notification fails, an error MUST be logged. The mutant skips the log.
	incidentID := uuid.New()
	monitorID := uuid.New()
	logger, logBuf := testLogger()

	incident := &domain.Incident{ID: incidentID, MonitorID: monitorID, Status: domain.IncidentStatusOpen}
	incidentRepo := &mocks.MockIncidentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return incident, nil
		},
		ResolveFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error { return nil },
	}
	notifier := &mocks.MockNotifier{
		NotifyIncidentResolvedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
			return errors.New("slack down")
		},
	}

	svc := services.NewIncidentService(incidentRepo, monitorRepo, &mocks.MockAgentRepository{}, &mocks.MockAlertChannelRepository{}, notifier, &mocks.MockTransactor{}, logger)

	err := svc.ResolveIncident(context.Background(), incidentID)
	require.NoError(t, err)
	assert.Contains(t, logBuf.String(), "global notification failed", "error must be logged when notification fails")
}

// --- CreateIncidentIfNeeded ---

func TestCreateIncidentIfNeeded_New(t *testing.T) {
	monitorID := uuid.New()
	created := false
	notified := false

	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil // No existing incident
		},
		CreateFn: func(_ context.Context, _ *domain.Incident) error {
			created = true
			return nil
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID, Name: "Test"}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, status domain.MonitorStatus) error {
			assert.Equal(t, domain.MonitorStatusDown, status)
			return nil
		},
	}
	notifier := &mocks.MockNotifier{
		NotifyIncidentOpenedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
			notified = true
			return nil
		},
	}

	svc := newTestIncidentService(incidentRepo, monitorRepo, notifier, &mocks.MockTransactor{})

	incident, err := svc.CreateIncidentIfNeeded(context.Background(), monitorID)

	require.NoError(t, err)
	assert.NotNil(t, incident)
	assert.True(t, created)
	assert.True(t, notified)
}

func TestCreateIncidentIfNeeded_AlreadyOpen(t *testing.T) {
	monitorID := uuid.New()
	existingIncident := domain.NewIncident(monitorID)

	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return existingIncident, nil
		},
		CreateFn: func(_ context.Context, _ *domain.Incident) error {
			t.Fatal("should not create new incident when one is already open")
			return nil
		},
	}

	svc := newTestIncidentService(incidentRepo, &mocks.MockMonitorRepository{}, &mocks.MockNotifier{}, &mocks.MockTransactor{})

	incident, err := svc.CreateIncidentIfNeeded(context.Background(), monitorID)

	require.NoError(t, err)
	assert.Equal(t, existingIncident.ID, incident.ID)
}

func TestCreateIncidentIfNeeded_MonitorNotFound(t *testing.T) {
	monitorID := uuid.New()

	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return nil, nil
		},
	}

	svc := newTestIncidentService(incidentRepo, monitorRepo, &mocks.MockNotifier{}, &mocks.MockTransactor{})

	incident, err := svc.CreateIncidentIfNeeded(context.Background(), monitorID)

	assert.Nil(t, incident)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "monitor not found")
}

func TestCreateIncidentIfNeeded_TransactionFails(t *testing.T) {
	monitorID := uuid.New()
	txErr := errors.New("tx failed")

	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID}, nil
		},
	}
	transactor := &mocks.MockTransactor{
		WithTransactionFn: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	}

	svc := newTestIncidentService(incidentRepo, monitorRepo, &mocks.MockNotifier{}, transactor)

	incident, err := svc.CreateIncidentIfNeeded(context.Background(), monitorID)

	assert.Nil(t, incident)
	assert.ErrorIs(t, err, txErr)
}

func TestCreateIncidentIfNeeded_NotificationCalledWithCorrectArgs(t *testing.T) {
	// Mutant survivor: incident_service.go:176 — negating notifyErr != nil
	// Verify notification is called with the correct incident and monitor.
	monitorID := uuid.New()
	var notifiedIncident *domain.Incident
	var notifiedMonitor *domain.Monitor

	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil
		},
		CreateFn: func(_ context.Context, _ *domain.Incident) error { return nil },
	}
	monitor := &domain.Monitor{ID: monitorID, Name: "Test Monitor"}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return monitor, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error { return nil },
	}
	notifier := &mocks.MockNotifier{
		NotifyIncidentOpenedFn: func(_ context.Context, inc *domain.Incident, mon *domain.Monitor) error {
			notifiedIncident = inc
			notifiedMonitor = mon
			return nil
		},
	}

	svc := newTestIncidentService(incidentRepo, monitorRepo, notifier, &mocks.MockTransactor{})

	incident, err := svc.CreateIncidentIfNeeded(context.Background(), monitorID)

	require.NoError(t, err)
	require.NotNil(t, notifiedIncident, "notification must be called")
	assert.Equal(t, incident.ID, notifiedIncident.ID, "notification should receive the created incident")
	assert.Equal(t, monitor.ID, notifiedMonitor.ID, "notification should receive the correct monitor")
}

func TestCreateIncidentIfNeeded_NotificationFails_LogsError(t *testing.T) {
	// Mutant survivor: incident_service.go:176:85 — notifyErr != nil negated
	// When notification fails, an error MUST be logged. The mutant skips the log.
	monitorID := uuid.New()
	logger, logBuf := testLogger()

	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil
		},
		CreateFn: func(_ context.Context, _ *domain.Incident) error { return nil },
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error { return nil },
	}
	notifier := &mocks.MockNotifier{
		NotifyIncidentOpenedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
			return errors.New("discord down")
		},
	}

	svc := services.NewIncidentService(incidentRepo, monitorRepo, &mocks.MockAgentRepository{}, &mocks.MockAlertChannelRepository{}, notifier, &mocks.MockTransactor{}, logger)

	incident, err := svc.CreateIncidentIfNeeded(context.Background(), monitorID)
	require.NoError(t, err)
	assert.NotNil(t, incident)
	assert.Contains(t, logBuf.String(), "global notification failed", "error must be logged when notification fails")
}

func TestCreateIncidentIfNeeded_NotificationFails_StillSucceeds(t *testing.T) {
	monitorID := uuid.New()

	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil
		},
		CreateFn: func(_ context.Context, _ *domain.Incident) error { return nil },
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error { return nil },
	}
	notifier := &mocks.MockNotifier{
		NotifyIncidentOpenedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
			return errors.New("discord down")
		},
	}

	svc := newTestIncidentService(incidentRepo, monitorRepo, notifier, &mocks.MockTransactor{})

	incident, err := svc.CreateIncidentIfNeeded(context.Background(), monitorID)

	// Should succeed despite notification failure
	require.NoError(t, err)
	assert.NotNil(t, incident)
}
