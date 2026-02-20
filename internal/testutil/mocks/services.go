package mocks

import (
	"context"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// Compile-time interface checks.
var (
	_ ports.UserAuthService  = (*MockUserAuthService)(nil)
	_ ports.AgentAuthService = (*MockAgentAuthService)(nil)
	_ ports.MonitorService   = (*MockMonitorService)(nil)
	_ ports.IncidentService  = (*MockIncidentService)(nil)
	_ ports.Notifier         = (*MockNotifier)(nil)
	_ ports.NotifierFactory  = (*MockNotifierFactory)(nil)
)

// MockUserAuthService is a mock implementation of ports.UserAuthService.
type MockUserAuthService struct {
	RegisterFn func(ctx context.Context, email, password string) (*domain.User, error)
	LoginFn    func(ctx context.Context, email, password string) (*domain.User, error)
}

func (m *MockUserAuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	if m.RegisterFn != nil {
		return m.RegisterFn(ctx, email, password)
	}
	return nil, nil
}

func (m *MockUserAuthService) Login(ctx context.Context, email, password string) (*domain.User, error) {
	if m.LoginFn != nil {
		return m.LoginFn(ctx, email, password)
	}
	return nil, nil
}

// MockAgentAuthService is a mock implementation of ports.AgentAuthService.
type MockAgentAuthService struct {
	ValidateAPIKeyFn func(ctx context.Context, apiKey string) (*domain.Agent, error)
	CreateAgentFn    func(ctx context.Context, userID string, name string) (*domain.Agent, string, error)
}

func (m *MockAgentAuthService) ValidateAPIKey(ctx context.Context, apiKey string) (*domain.Agent, error) {
	if m.ValidateAPIKeyFn != nil {
		return m.ValidateAPIKeyFn(ctx, apiKey)
	}
	return nil, nil
}

func (m *MockAgentAuthService) CreateAgent(ctx context.Context, userID string, name string) (*domain.Agent, string, error) {
	if m.CreateAgentFn != nil {
		return m.CreateAgentFn(ctx, userID, name)
	}
	return nil, "", nil
}

// MockMonitorService is a mock implementation of ports.MonitorService.
type MockMonitorService struct {
	CreateMonitorFn    func(ctx context.Context, userID uuid.UUID, agentID uuid.UUID, name string, monitorType domain.MonitorType, target string) (*domain.Monitor, error)
	GetMonitorFn       func(ctx context.Context, id uuid.UUID) (*domain.Monitor, error)
	GetMonitorsByAgentFn func(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error)
	UpdateMonitorFn    func(ctx context.Context, monitor *domain.Monitor) error
	DeleteMonitorFn    func(ctx context.Context, id uuid.UUID) error
	ProcessHeartbeatFn func(ctx context.Context, heartbeat *domain.Heartbeat) error
}

func (m *MockMonitorService) CreateMonitor(ctx context.Context, userID uuid.UUID, agentID uuid.UUID, name string, monitorType domain.MonitorType, target string) (*domain.Monitor, error) {
	if m.CreateMonitorFn != nil {
		return m.CreateMonitorFn(ctx, userID, agentID, name, monitorType, target)
	}
	return nil, nil
}

func (m *MockMonitorService) GetMonitor(ctx context.Context, id uuid.UUID) (*domain.Monitor, error) {
	if m.GetMonitorFn != nil {
		return m.GetMonitorFn(ctx, id)
	}
	return nil, nil
}

func (m *MockMonitorService) GetMonitorsByAgent(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error) {
	if m.GetMonitorsByAgentFn != nil {
		return m.GetMonitorsByAgentFn(ctx, agentID)
	}
	return nil, nil
}

func (m *MockMonitorService) UpdateMonitor(ctx context.Context, monitor *domain.Monitor) error {
	if m.UpdateMonitorFn != nil {
		return m.UpdateMonitorFn(ctx, monitor)
	}
	return nil
}

func (m *MockMonitorService) DeleteMonitor(ctx context.Context, id uuid.UUID) error {
	if m.DeleteMonitorFn != nil {
		return m.DeleteMonitorFn(ctx, id)
	}
	return nil
}

func (m *MockMonitorService) ProcessHeartbeat(ctx context.Context, heartbeat *domain.Heartbeat) error {
	if m.ProcessHeartbeatFn != nil {
		return m.ProcessHeartbeatFn(ctx, heartbeat)
	}
	return nil
}

// MockIncidentService is a mock implementation of ports.IncidentService.
type MockIncidentService struct {
	GetIncidentFn            func(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	GetActiveIncidentsFn     func(ctx context.Context) ([]*domain.Incident, error)
	GetResolvedIncidentsFn   func(ctx context.Context) ([]*domain.Incident, error)
	GetAllIncidentsFn        func(ctx context.Context) ([]*domain.Incident, error)
	GetIncidentsByMonitorFn  func(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error)
	AcknowledgeIncidentFn    func(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ResolveIncidentFn        func(ctx context.Context, id uuid.UUID) error
	CreateIncidentIfNeededFn func(ctx context.Context, monitorID uuid.UUID) (*domain.Incident, error)
}

func (m *MockIncidentService) GetIncident(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	if m.GetIncidentFn != nil {
		return m.GetIncidentFn(ctx, id)
	}
	return nil, nil
}

func (m *MockIncidentService) GetActiveIncidents(ctx context.Context) ([]*domain.Incident, error) {
	if m.GetActiveIncidentsFn != nil {
		return m.GetActiveIncidentsFn(ctx)
	}
	return nil, nil
}

func (m *MockIncidentService) GetResolvedIncidents(ctx context.Context) ([]*domain.Incident, error) {
	if m.GetResolvedIncidentsFn != nil {
		return m.GetResolvedIncidentsFn(ctx)
	}
	return nil, nil
}

func (m *MockIncidentService) GetAllIncidents(ctx context.Context) ([]*domain.Incident, error) {
	if m.GetAllIncidentsFn != nil {
		return m.GetAllIncidentsFn(ctx)
	}
	return nil, nil
}

func (m *MockIncidentService) GetIncidentsByMonitor(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error) {
	if m.GetIncidentsByMonitorFn != nil {
		return m.GetIncidentsByMonitorFn(ctx, monitorID)
	}
	return nil, nil
}

func (m *MockIncidentService) AcknowledgeIncident(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if m.AcknowledgeIncidentFn != nil {
		return m.AcknowledgeIncidentFn(ctx, id, userID)
	}
	return nil
}

func (m *MockIncidentService) ResolveIncident(ctx context.Context, id uuid.UUID) error {
	if m.ResolveIncidentFn != nil {
		return m.ResolveIncidentFn(ctx, id)
	}
	return nil
}

func (m *MockIncidentService) CreateIncidentIfNeeded(ctx context.Context, monitorID uuid.UUID) (*domain.Incident, error) {
	if m.CreateIncidentIfNeededFn != nil {
		return m.CreateIncidentIfNeededFn(ctx, monitorID)
	}
	return nil, nil
}

// MockNotifier is a mock implementation of ports.Notifier.
type MockNotifier struct {
	NotifyIncidentOpenedFn   func(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
	NotifyIncidentResolvedFn func(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
}

func (m *MockNotifier) NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	if m.NotifyIncidentOpenedFn != nil {
		return m.NotifyIncidentOpenedFn(ctx, incident, monitor)
	}
	return nil
}

func (m *MockNotifier) NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	if m.NotifyIncidentResolvedFn != nil {
		return m.NotifyIncidentResolvedFn(ctx, incident, monitor)
	}
	return nil
}

// MockNotifierFactory is a mock implementation of ports.NotifierFactory.
type MockNotifierFactory struct {
	BuildFromChannelFn func(channel *domain.AlertChannel) (ports.Notifier, error)
}

func (m *MockNotifierFactory) BuildFromChannel(channel *domain.AlertChannel) (ports.Notifier, error) {
	if m.BuildFromChannelFn != nil {
		return m.BuildFromChannelFn(channel)
	}
	return &MockNotifier{}, nil
}
