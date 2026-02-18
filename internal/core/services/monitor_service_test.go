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

func newTestMonitorService(
	monitorRepo *mocks.MockMonitorRepository,
	heartbeatRepo *mocks.MockHeartbeatRepository,
	incidentRepo *mocks.MockIncidentRepository,
	incidentSvc *mocks.MockIncidentService,
) *services.MonitorService {
	logger := slog.Default()
	return services.NewMonitorService(monitorRepo, heartbeatRepo, incidentRepo, incidentSvc, &mocks.MockUserRepository{}, &mocks.MockUsageEventRepository{}, logger)
}

// --- NewMonitorService nil logger ---

func TestNewMonitorService_NilLogger(t *testing.T) {
	// Mutant survivor: monitor_service.go:35 — if logger == nil
	// If the nil guard is negated, logger stays nil and will PANIC on log call.
	// Trigger the handleRecovery path where UpdateStatus fails → logger.Warn is called.
	monitorID := uuid.New()

	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error { return nil },
	}
	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil // No open incident → takes UpdateStatus path
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error {
			return errors.New("status update failed") // Forces logger.Warn call
		},
	}

	svc := services.NewMonitorService(monitorRepo, heartbeatRepo, incidentRepo, &mocks.MockIncidentService{}, &mocks.MockUserRepository{}, &mocks.MockUsageEventRepository{}, nil)
	require.NotNil(t, svc)

	// This triggers logger.Warn — if nil guard was mutated, this panics
	hb := domain.NewSuccessHeartbeat(monitorID, uuid.New(), 50)
	err := svc.ProcessHeartbeat(context.Background(), hb)
	require.NoError(t, err)
}

// --- CreateMonitor ---

func TestCreateMonitor_Success(t *testing.T) {
	userID := uuid.New()
	var savedMonitor *domain.Monitor
	userRepo := &mocks.MockUserRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return &domain.User{ID: userID, Plan: domain.PlanBeta}, nil
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		CountByUserIDFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFn: func(_ context.Context, m *domain.Monitor) error {
			savedMonitor = m
			return nil
		},
	}
	svc := services.NewMonitorService(monitorRepo, &mocks.MockHeartbeatRepository{}, &mocks.MockIncidentRepository{}, &mocks.MockIncidentService{}, userRepo, &mocks.MockUsageEventRepository{}, slog.Default())

	agentID := uuid.New()
	monitor, err := svc.CreateMonitor(context.Background(), userID, agentID, "Test HTTP", domain.MonitorTypeHTTP, "https://example.com")

	require.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.Equal(t, "Test HTTP", monitor.Name)
	assert.Equal(t, domain.MonitorTypeHTTP, monitor.Type)
	assert.Equal(t, "https://example.com", monitor.Target)
	assert.Equal(t, agentID, monitor.AgentID)
	assert.Equal(t, savedMonitor, monitor)
}

func TestCreateMonitor_RepoError(t *testing.T) {
	userID := uuid.New()
	repoErr := errors.New("db error")
	userRepo := &mocks.MockUserRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return &domain.User{ID: userID, Plan: domain.PlanBeta}, nil
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		CreateFn: func(_ context.Context, _ *domain.Monitor) error {
			return repoErr
		},
	}
	svc := services.NewMonitorService(monitorRepo, &mocks.MockHeartbeatRepository{}, &mocks.MockIncidentRepository{}, &mocks.MockIncidentService{}, userRepo, &mocks.MockUsageEventRepository{}, slog.Default())

	monitor, err := svc.CreateMonitor(context.Background(), userID, uuid.New(), "Test", domain.MonitorTypePing, "8.8.8.8")

	assert.Nil(t, monitor)
	assert.ErrorIs(t, err, repoErr)
}

// --- GetMonitorsByAgent ---

func TestGetMonitorsByAgent_Success(t *testing.T) {
	agentID := uuid.New()
	expected := []*domain.Monitor{
		{ID: uuid.New(), AgentID: agentID, Name: "Mon1"},
		{ID: uuid.New(), AgentID: agentID, Name: "Mon2"},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		GetByAgentIDFn: func(_ context.Context, id uuid.UUID) ([]*domain.Monitor, error) {
			assert.Equal(t, agentID, id)
			return expected, nil
		},
	}
	svc := newTestMonitorService(monitorRepo, &mocks.MockHeartbeatRepository{}, &mocks.MockIncidentRepository{}, &mocks.MockIncidentService{})

	monitors, err := svc.GetMonitorsByAgent(context.Background(), agentID)

	require.NoError(t, err)
	assert.Len(t, monitors, 2)
}

// --- DeleteMonitor ---

func TestDeleteMonitor_Success(t *testing.T) {
	monitorID := uuid.New()
	deleted := false
	monitorRepo := &mocks.MockMonitorRepository{
		DeleteFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, monitorID, id)
			deleted = true
			return nil
		},
	}
	svc := newTestMonitorService(monitorRepo, &mocks.MockHeartbeatRepository{}, &mocks.MockIncidentRepository{}, &mocks.MockIncidentService{})

	err := svc.DeleteMonitor(context.Background(), monitorID)

	require.NoError(t, err)
	assert.True(t, deleted)
}

// --- ProcessHeartbeat (3-Strike Rule) ---

func TestProcessHeartbeat_Success_NoIncident(t *testing.T) {
	monitorID := uuid.New()
	statusUpdated := false

	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error { return nil },
	}
	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil // No open incident
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		UpdateStatusFn: func(_ context.Context, id uuid.UUID, status domain.MonitorStatus) error {
			assert.Equal(t, monitorID, id)
			assert.Equal(t, domain.MonitorStatusUp, status)
			statusUpdated = true
			return nil
		},
	}

	svc := newTestMonitorService(monitorRepo, heartbeatRepo, incidentRepo, &mocks.MockIncidentService{})

	hb := domain.NewSuccessHeartbeat(monitorID, uuid.New(), 50)
	err := svc.ProcessHeartbeat(context.Background(), hb)

	require.NoError(t, err)
	assert.True(t, statusUpdated)
}

func TestProcessHeartbeat_Success_ResolvesIncident(t *testing.T) {
	monitorID := uuid.New()
	incidentID := uuid.New()
	resolved := false

	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error { return nil },
	}
	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return &domain.Incident{
				ID:        incidentID,
				MonitorID: monitorID,
				Status:    domain.IncidentStatusOpen,
			}, nil
		},
	}
	incidentSvc := &mocks.MockIncidentService{
		ResolveIncidentFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, incidentID, id)
			resolved = true
			return nil
		},
	}

	svc := newTestMonitorService(&mocks.MockMonitorRepository{}, heartbeatRepo, incidentRepo, incidentSvc)

	hb := domain.NewSuccessHeartbeat(monitorID, uuid.New(), 50)
	err := svc.ProcessHeartbeat(context.Background(), hb)

	require.NoError(t, err)
	assert.True(t, resolved)
}

func TestProcessHeartbeat_SingleFailure_NoIncident(t *testing.T) {
	monitorID := uuid.New()

	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error { return nil },
		GetByMonitorIDFn: func(_ context.Context, _ uuid.UUID, _ int) ([]*domain.Heartbeat, error) {
			// Only 1 heartbeat (the one we just created)
			return []*domain.Heartbeat{
				domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "timeout"),
			}, nil
		},
	}
	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil // No open incident
		},
	}
	incidentSvc := &mocks.MockIncidentService{
		CreateIncidentIfNeededFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			t.Fatal("should not create incident for single failure")
			return nil, nil
		},
	}

	svc := newTestMonitorService(&mocks.MockMonitorRepository{}, heartbeatRepo, incidentRepo, incidentSvc)

	hb := domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "timeout")
	err := svc.ProcessHeartbeat(context.Background(), hb)

	require.NoError(t, err)
}

func TestProcessHeartbeat_TwoFailures_NoIncident(t *testing.T) {
	monitorID := uuid.New()

	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error { return nil },
		GetByMonitorIDFn: func(_ context.Context, _ uuid.UUID, _ int) ([]*domain.Heartbeat, error) {
			return []*domain.Heartbeat{
				domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err"),
				domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err"),
			}, nil
		},
	}
	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil
		},
	}
	incidentSvc := &mocks.MockIncidentService{
		CreateIncidentIfNeededFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			t.Fatal("should not create incident for only 2 failures")
			return nil, nil
		},
	}

	svc := newTestMonitorService(&mocks.MockMonitorRepository{}, heartbeatRepo, incidentRepo, incidentSvc)

	hb := domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err")
	err := svc.ProcessHeartbeat(context.Background(), hb)

	require.NoError(t, err)
}

func TestProcessHeartbeat_ThreeConsecutiveFailures_CreatesIncident(t *testing.T) {
	monitorID := uuid.New()
	incidentCreated := false

	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error { return nil },
		GetByMonitorIDFn: func(_ context.Context, _ uuid.UUID, _ int) ([]*domain.Heartbeat, error) {
			return []*domain.Heartbeat{
				domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err"),
				domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err"),
				domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err"),
			}, nil
		},
	}
	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil
		},
	}
	incidentSvc := &mocks.MockIncidentService{
		CreateIncidentIfNeededFn: func(_ context.Context, id uuid.UUID) (*domain.Incident, error) {
			assert.Equal(t, monitorID, id)
			incidentCreated = true
			return domain.NewIncident(monitorID), nil
		},
	}

	svc := newTestMonitorService(&mocks.MockMonitorRepository{}, heartbeatRepo, incidentRepo, incidentSvc)

	hb := domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err")
	err := svc.ProcessHeartbeat(context.Background(), hb)

	require.NoError(t, err)
	assert.True(t, incidentCreated)
}

func TestProcessHeartbeat_FailuresNotConsecutive_NoIncident(t *testing.T) {
	monitorID := uuid.New()

	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error { return nil },
		GetByMonitorIDFn: func(_ context.Context, _ uuid.UUID, _ int) ([]*domain.Heartbeat, error) {
			// 2 failures with a success in between
			return []*domain.Heartbeat{
				domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err"),
				domain.NewSuccessHeartbeat(monitorID, uuid.New(), 50),
				domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err"),
			}, nil
		},
	}
	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil
		},
	}
	incidentSvc := &mocks.MockIncidentService{
		CreateIncidentIfNeededFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			t.Fatal("should not create incident when failures are not consecutive")
			return nil, nil
		},
	}

	svc := newTestMonitorService(&mocks.MockMonitorRepository{}, heartbeatRepo, incidentRepo, incidentSvc)

	hb := domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err")
	err := svc.ProcessHeartbeat(context.Background(), hb)

	require.NoError(t, err)
}

func TestProcessHeartbeat_AlreadyOpenIncident_NoNew(t *testing.T) {
	monitorID := uuid.New()

	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error { return nil },
	}
	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return domain.NewIncident(monitorID), nil // Existing open incident
		},
	}
	incidentSvc := &mocks.MockIncidentService{
		CreateIncidentIfNeededFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			t.Fatal("should not create another incident when one is already open")
			return nil, nil
		},
	}

	svc := newTestMonitorService(&mocks.MockMonitorRepository{}, heartbeatRepo, incidentRepo, incidentSvc)

	hb := domain.NewFailureHeartbeat(monitorID, uuid.New(), domain.HeartbeatStatusDown, "err")
	err := svc.ProcessHeartbeat(context.Background(), hb)

	require.NoError(t, err)
}

func TestProcessHeartbeat_Success_UpdateStatusError_LogsWarning(t *testing.T) {
	// Mutant survivor: monitor_service.go:120:85 — err != nil negated after UpdateStatus
	// When UpdateStatus fails, a warning MUST be logged. The mutant skips the log.
	monitorID := uuid.New()
	logBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error { return nil },
	}
	incidentRepo := &mocks.MockIncidentRepository{
		GetOpenByMonitorIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Incident, error) {
			return nil, nil // No open incident — takes the UpdateStatus path
		},
	}
	monitorRepo := &mocks.MockMonitorRepository{
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.MonitorStatus) error {
			return errors.New("status update failed")
		},
	}

	svc := services.NewMonitorService(monitorRepo, heartbeatRepo, incidentRepo, &mocks.MockIncidentService{}, &mocks.MockUserRepository{}, &mocks.MockUsageEventRepository{}, logger)

	hb := domain.NewSuccessHeartbeat(monitorID, uuid.New(), 50)
	err := svc.ProcessHeartbeat(context.Background(), hb)

	require.NoError(t, err, "UpdateStatus error should be swallowed, not propagated")
	assert.Contains(t, logBuf.String(), "failed to update monitor status", "warning must be logged when UpdateStatus fails")
}

func TestProcessHeartbeat_StoreError_Propagates(t *testing.T) {
	storeErr := errors.New("storage failure")
	heartbeatRepo := &mocks.MockHeartbeatRepository{
		CreateFn: func(_ context.Context, _ *domain.Heartbeat) error {
			return storeErr
		},
	}

	svc := newTestMonitorService(&mocks.MockMonitorRepository{}, heartbeatRepo, &mocks.MockIncidentRepository{}, &mocks.MockIncidentService{})

	hb := domain.NewSuccessHeartbeat(uuid.New(), uuid.New(), 50)
	err := svc.ProcessHeartbeat(context.Background(), hb)

	assert.ErrorIs(t, err, storeErr)
}
