CREATE TABLE maintenance_windows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id VARCHAR(255) NOT NULL DEFAULT 'default',
    CONSTRAINT chk_mw_range CHECK (ends_at > starts_at)
);

CREATE INDEX idx_mw_agent_active ON maintenance_windows(agent_id, starts_at, ends_at);
CREATE INDEX idx_mw_tenant ON maintenance_windows(tenant_id);
