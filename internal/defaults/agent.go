package defaults

import (
	"context"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/registry"
)

const moduleAgentManager = "agent_manager"

var (
	_ registry.Module   = (*agentModule)(nil)
	_ ports.AgentManager = (*agentModule)(nil)
)

// agentModule wraps AgentAuthService and AgentRepository for agent lifecycle.
type agentModule struct {
	auth ports.AgentAuthService
	repo ports.AgentRepository
}

func newAgentModule(auth ports.AgentAuthService, repo ports.AgentRepository) *agentModule {
	return &agentModule{auth: auth, repo: repo}
}

func (m *agentModule) Name() string                    { return moduleAgentManager }
func (m *agentModule) Init(_ context.Context) error    { return nil }
func (m *agentModule) Health(_ context.Context) error   { return nil }
func (m *agentModule) Shutdown(_ context.Context) error { return nil }

func (m *agentModule) CreateAgent(ctx context.Context, userID string, name string) (*domain.Agent, string, error) {
	return m.auth.CreateAgent(ctx, userID, name)
}

func (m *agentModule) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	return m.repo.Delete(ctx, id)
}

func (m *agentModule) ListAgents(ctx context.Context, userID uuid.UUID) ([]*domain.Agent, error) {
	return m.repo.GetByUserID(ctx, userID)
}

func (m *agentModule) ValidateAPIKey(ctx context.Context, apiKey string) (*domain.Agent, error) {
	return m.auth.ValidateAPIKey(ctx, apiKey)
}
