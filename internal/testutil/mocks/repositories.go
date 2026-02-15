package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// Compile-time interface checks.
var (
	_ ports.UserRepository       = (*MockUserRepository)(nil)
	_ ports.AgentRepository      = (*MockAgentRepository)(nil)
	_ ports.MonitorRepository    = (*MockMonitorRepository)(nil)
	_ ports.IncidentRepository   = (*MockIncidentRepository)(nil)
	_ ports.HeartbeatRepository  = (*MockHeartbeatRepository)(nil)
	_ ports.UsageEventRepository = (*MockUsageEventRepository)(nil)
	_ ports.WaitlistRepository   = (*MockWaitlistRepository)(nil)
	_ ports.Transactor           = (*MockTransactor)(nil)
)

// MockUserRepository is a mock implementation of ports.UserRepository.
type MockUserRepository struct {
	CreateFn             func(ctx context.Context, user *domain.User) error
	GetByIDFn            func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmailFn         func(ctx context.Context, email string) (*domain.User, error)
	UpdateFn             func(ctx context.Context, user *domain.User) error
	DeleteFn             func(ctx context.Context, id uuid.UUID) error
	ExistsByEmailFn      func(ctx context.Context, email string) (bool, error)
	CountFn              func(ctx context.Context) (int, error)
	CountByPlanFn        func(ctx context.Context) (map[domain.Plan]int, error)
	GetUsersNearLimitsFn func(ctx context.Context) ([]ports.UserUsageSummary, error)
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.ExistsByEmailFn != nil {
		return m.ExistsByEmailFn(ctx, email)
	}
	return false, nil
}

func (m *MockUserRepository) Count(ctx context.Context) (int, error) {
	if m.CountFn != nil {
		return m.CountFn(ctx)
	}
	return 0, nil
}

func (m *MockUserRepository) CountByPlan(ctx context.Context) (map[domain.Plan]int, error) {
	if m.CountByPlanFn != nil {
		return m.CountByPlanFn(ctx)
	}
	return nil, nil
}

func (m *MockUserRepository) GetUsersNearLimits(ctx context.Context) ([]ports.UserUsageSummary, error) {
	if m.GetUsersNearLimitsFn != nil {
		return m.GetUsersNearLimitsFn(ctx)
	}
	return nil, nil
}

// MockAgentRepository is a mock implementation of ports.AgentRepository.
type MockAgentRepository struct {
	CreateFn          func(ctx context.Context, agent *domain.Agent) error
	GetByIDFn         func(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	GetByUserIDFn     func(ctx context.Context, userID uuid.UUID) ([]*domain.Agent, error)
	UpdateFn          func(ctx context.Context, agent *domain.Agent) error
	DeleteFn          func(ctx context.Context, id uuid.UUID) error
	UpdateStatusFn    func(ctx context.Context, id uuid.UUID, status domain.AgentStatus) error
	UpdateLastSeenFn  func(ctx context.Context, id uuid.UUID, lastSeen time.Time) error
	CountByUserIDFn   func(ctx context.Context, userID uuid.UUID) (int, error)
}

func (m *MockAgentRepository) Create(ctx context.Context, agent *domain.Agent) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, agent)
	}
	return nil
}

func (m *MockAgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockAgentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Agent, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (m *MockAgentRepository) Update(ctx context.Context, agent *domain.Agent) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, agent)
	}
	return nil
}

func (m *MockAgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *MockAgentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AgentStatus) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *MockAgentRepository) UpdateLastSeen(ctx context.Context, id uuid.UUID, lastSeen time.Time) error {
	if m.UpdateLastSeenFn != nil {
		return m.UpdateLastSeenFn(ctx, id, lastSeen)
	}
	return nil
}

func (m *MockAgentRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.CountByUserIDFn != nil {
		return m.CountByUserIDFn(ctx, userID)
	}
	return 0, nil
}

// MockMonitorRepository is a mock implementation of ports.MonitorRepository.
type MockMonitorRepository struct {
	CreateFn              func(ctx context.Context, monitor *domain.Monitor) error
	GetByIDFn             func(ctx context.Context, id uuid.UUID) (*domain.Monitor, error)
	GetByAgentIDFn        func(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error)
	GetEnabledByAgentIDFn func(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error)
	UpdateFn              func(ctx context.Context, monitor *domain.Monitor) error
	DeleteFn              func(ctx context.Context, id uuid.UUID) error
	UpdateStatusFn        func(ctx context.Context, id uuid.UUID, status domain.MonitorStatus) error
	CountByUserIDFn       func(ctx context.Context, userID uuid.UUID) (int, error)
}

func (m *MockMonitorRepository) Create(ctx context.Context, monitor *domain.Monitor) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, monitor)
	}
	return nil
}

func (m *MockMonitorRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Monitor, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockMonitorRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error) {
	if m.GetByAgentIDFn != nil {
		return m.GetByAgentIDFn(ctx, agentID)
	}
	return nil, nil
}

func (m *MockMonitorRepository) GetEnabledByAgentID(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error) {
	if m.GetEnabledByAgentIDFn != nil {
		return m.GetEnabledByAgentIDFn(ctx, agentID)
	}
	return nil, nil
}

func (m *MockMonitorRepository) Update(ctx context.Context, monitor *domain.Monitor) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, monitor)
	}
	return nil
}

func (m *MockMonitorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *MockMonitorRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MonitorStatus) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *MockMonitorRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.CountByUserIDFn != nil {
		return m.CountByUserIDFn(ctx, userID)
	}
	return 0, nil
}

// MockIncidentRepository is a mock implementation of ports.IncidentRepository.
type MockIncidentRepository struct {
	CreateFn               func(ctx context.Context, incident *domain.Incident) error
	GetByIDFn              func(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	GetByMonitorIDFn       func(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error)
	GetOpenByMonitorIDFn   func(ctx context.Context, monitorID uuid.UUID) (*domain.Incident, error)
	GetActiveIncidentsFn   func(ctx context.Context) ([]*domain.Incident, error)
	GetResolvedIncidentsFn func(ctx context.Context) ([]*domain.Incident, error)
	GetAllIncidentsFn      func(ctx context.Context) ([]*domain.Incident, error)
	UpdateFn               func(ctx context.Context, incident *domain.Incident) error
	AcknowledgeFn          func(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ResolveFn              func(ctx context.Context, id uuid.UUID) error
}

func (m *MockIncidentRepository) Create(ctx context.Context, incident *domain.Incident) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, incident)
	}
	return nil
}

func (m *MockIncidentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockIncidentRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error) {
	if m.GetByMonitorIDFn != nil {
		return m.GetByMonitorIDFn(ctx, monitorID)
	}
	return nil, nil
}

func (m *MockIncidentRepository) GetOpenByMonitorID(ctx context.Context, monitorID uuid.UUID) (*domain.Incident, error) {
	if m.GetOpenByMonitorIDFn != nil {
		return m.GetOpenByMonitorIDFn(ctx, monitorID)
	}
	return nil, nil
}

func (m *MockIncidentRepository) GetActiveIncidents(ctx context.Context) ([]*domain.Incident, error) {
	if m.GetActiveIncidentsFn != nil {
		return m.GetActiveIncidentsFn(ctx)
	}
	return nil, nil
}

func (m *MockIncidentRepository) GetResolvedIncidents(ctx context.Context) ([]*domain.Incident, error) {
	if m.GetResolvedIncidentsFn != nil {
		return m.GetResolvedIncidentsFn(ctx)
	}
	return nil, nil
}

func (m *MockIncidentRepository) GetAllIncidents(ctx context.Context) ([]*domain.Incident, error) {
	if m.GetAllIncidentsFn != nil {
		return m.GetAllIncidentsFn(ctx)
	}
	return nil, nil
}

func (m *MockIncidentRepository) Update(ctx context.Context, incident *domain.Incident) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, incident)
	}
	return nil
}

func (m *MockIncidentRepository) Acknowledge(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if m.AcknowledgeFn != nil {
		return m.AcknowledgeFn(ctx, id, userID)
	}
	return nil
}

func (m *MockIncidentRepository) Resolve(ctx context.Context, id uuid.UUID) error {
	if m.ResolveFn != nil {
		return m.ResolveFn(ctx, id)
	}
	return nil
}

// MockHeartbeatRepository is a mock implementation of ports.HeartbeatRepository.
type MockHeartbeatRepository struct {
	CreateFn               func(ctx context.Context, heartbeat *domain.Heartbeat) error
	CreateBatchFn          func(ctx context.Context, heartbeats []*domain.Heartbeat) error
	GetByMonitorIDFn       func(ctx context.Context, monitorID uuid.UUID, limit int) ([]*domain.Heartbeat, error)
	GetByMonitorIDInRangeFn func(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.Heartbeat, error)
	GetLatestByMonitorIDFn func(ctx context.Context, monitorID uuid.UUID) (*domain.Heartbeat, error)
	GetRecentFailuresFn    func(ctx context.Context, monitorID uuid.UUID, count int) ([]*domain.Heartbeat, error)
	DeleteOlderThanFn      func(ctx context.Context, before time.Time) (int64, error)
}

func (m *MockHeartbeatRepository) Create(ctx context.Context, heartbeat *domain.Heartbeat) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, heartbeat)
	}
	return nil
}

func (m *MockHeartbeatRepository) CreateBatch(ctx context.Context, heartbeats []*domain.Heartbeat) error {
	if m.CreateBatchFn != nil {
		return m.CreateBatchFn(ctx, heartbeats)
	}
	return nil
}

func (m *MockHeartbeatRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]*domain.Heartbeat, error) {
	if m.GetByMonitorIDFn != nil {
		return m.GetByMonitorIDFn(ctx, monitorID, limit)
	}
	return nil, nil
}

func (m *MockHeartbeatRepository) GetByMonitorIDInRange(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.Heartbeat, error) {
	if m.GetByMonitorIDInRangeFn != nil {
		return m.GetByMonitorIDInRangeFn(ctx, monitorID, from, to)
	}
	return nil, nil
}

func (m *MockHeartbeatRepository) GetLatestByMonitorID(ctx context.Context, monitorID uuid.UUID) (*domain.Heartbeat, error) {
	if m.GetLatestByMonitorIDFn != nil {
		return m.GetLatestByMonitorIDFn(ctx, monitorID)
	}
	return nil, nil
}

func (m *MockHeartbeatRepository) GetRecentFailures(ctx context.Context, monitorID uuid.UUID, count int) ([]*domain.Heartbeat, error) {
	if m.GetRecentFailuresFn != nil {
		return m.GetRecentFailuresFn(ctx, monitorID, count)
	}
	return nil, nil
}

func (m *MockHeartbeatRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	if m.DeleteOlderThanFn != nil {
		return m.DeleteOlderThanFn(ctx, before)
	}
	return 0, nil
}

// MockUsageEventRepository is a mock implementation of ports.UsageEventRepository.
type MockUsageEventRepository struct {
	CreateFn           func(ctx context.Context, event *domain.UsageEvent) error
	GetRecentByUserIDFn func(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.UsageEvent, error)
	GetRecentFn        func(ctx context.Context, limit int) ([]*domain.UsageEvent, error)
	CountByEventTypeFn func(ctx context.Context, eventType domain.EventType, since time.Time) (int, error)
}

func (m *MockUsageEventRepository) Create(ctx context.Context, event *domain.UsageEvent) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, event)
	}
	return nil
}

func (m *MockUsageEventRepository) GetRecentByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.UsageEvent, error) {
	if m.GetRecentByUserIDFn != nil {
		return m.GetRecentByUserIDFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *MockUsageEventRepository) GetRecent(ctx context.Context, limit int) ([]*domain.UsageEvent, error) {
	if m.GetRecentFn != nil {
		return m.GetRecentFn(ctx, limit)
	}
	return nil, nil
}

func (m *MockUsageEventRepository) CountByEventType(ctx context.Context, eventType domain.EventType, since time.Time) (int, error) {
	if m.CountByEventTypeFn != nil {
		return m.CountByEventTypeFn(ctx, eventType, since)
	}
	return 0, nil
}

// MockWaitlistRepository is a mock implementation of ports.WaitlistRepository.
type MockWaitlistRepository struct {
	CreateFn     func(ctx context.Context, signup *domain.WaitlistSignup) error
	GetByEmailFn func(ctx context.Context, email string) (*domain.WaitlistSignup, error)
	CountFn      func(ctx context.Context) (int, error)
}

func (m *MockWaitlistRepository) Create(ctx context.Context, signup *domain.WaitlistSignup) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, signup)
	}
	return nil
}

func (m *MockWaitlistRepository) GetByEmail(ctx context.Context, email string) (*domain.WaitlistSignup, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *MockWaitlistRepository) Count(ctx context.Context) (int, error) {
	if m.CountFn != nil {
		return m.CountFn(ctx)
	}
	return 0, nil
}

// MockTransactor is a mock implementation of ports.Transactor.
type MockTransactor struct {
	WithTransactionFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *MockTransactor) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.WithTransactionFn != nil {
		return m.WithTransactionFn(ctx, fn)
	}
	// Default: just execute the function without a real transaction
	return fn(ctx)
}
