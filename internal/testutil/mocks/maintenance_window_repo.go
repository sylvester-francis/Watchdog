package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// Compile-time interface check.
var _ ports.MaintenanceWindowRepository = (*MockMaintenanceWindowRepository)(nil)

// MockMaintenanceWindowRepository is a mock implementation of ports.MaintenanceWindowRepository.
type MockMaintenanceWindowRepository struct {
	CreateFn                      func(ctx context.Context, window *domain.MaintenanceWindow) error
	GetByIDFn                     func(ctx context.Context, id uuid.UUID) (*domain.MaintenanceWindow, error)
	GetByTenantFn                 func(ctx context.Context) ([]*domain.MaintenanceWindow, error)
	GetActiveByAgentIDFn          func(ctx context.Context, agentID uuid.UUID) (*domain.MaintenanceWindow, error)
	UpdateFn                      func(ctx context.Context, window *domain.MaintenanceWindow) error
	DeleteFn                      func(ctx context.Context, id uuid.UUID) error
	GetExpiredWithOfflineAgentsFn func(ctx context.Context) ([]*domain.MaintenanceWindow, error)
	GetExpiredRecurringFn         func(ctx context.Context) ([]*domain.MaintenanceWindow, error)
	DeleteExpiredFn               func(ctx context.Context, before time.Time) error
}

func (m *MockMaintenanceWindowRepository) Create(ctx context.Context, window *domain.MaintenanceWindow) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, window)
	}
	return nil
}

func (m *MockMaintenanceWindowRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MaintenanceWindow, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockMaintenanceWindowRepository) GetByTenant(ctx context.Context) ([]*domain.MaintenanceWindow, error) {
	if m.GetByTenantFn != nil {
		return m.GetByTenantFn(ctx)
	}
	return nil, nil
}

func (m *MockMaintenanceWindowRepository) GetActiveByAgentID(ctx context.Context, agentID uuid.UUID) (*domain.MaintenanceWindow, error) {
	if m.GetActiveByAgentIDFn != nil {
		return m.GetActiveByAgentIDFn(ctx, agentID)
	}
	return nil, nil
}

func (m *MockMaintenanceWindowRepository) Update(ctx context.Context, window *domain.MaintenanceWindow) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, window)
	}
	return nil
}

func (m *MockMaintenanceWindowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *MockMaintenanceWindowRepository) GetExpiredWithOfflineAgents(ctx context.Context) ([]*domain.MaintenanceWindow, error) {
	if m.GetExpiredWithOfflineAgentsFn != nil {
		return m.GetExpiredWithOfflineAgentsFn(ctx)
	}
	return nil, nil
}

func (m *MockMaintenanceWindowRepository) GetExpiredRecurring(ctx context.Context) ([]*domain.MaintenanceWindow, error) {
	if m.GetExpiredRecurringFn != nil {
		return m.GetExpiredRecurringFn(ctx)
	}
	return nil, nil
}

func (m *MockMaintenanceWindowRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	if m.DeleteExpiredFn != nil {
		return m.DeleteExpiredFn(ctx, before)
	}
	return nil
}
