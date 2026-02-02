-- Create monitors table
CREATE TABLE monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    target VARCHAR(500) NOT NULL,
    interval_seconds INT NOT NULL DEFAULT 30,
    timeout_seconds INT NOT NULL DEFAULT 10,
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for agent's monitors lookup
CREATE INDEX idx_monitors_agent_id ON monitors(agent_id);

-- Index for enabled monitors (used in task distribution)
CREATE INDEX idx_monitors_enabled ON monitors(agent_id, enabled) WHERE enabled = true;

-- Index for status filtering
CREATE INDEX idx_monitors_status ON monitors(status);

-- Constraint for valid monitor types
ALTER TABLE monitors ADD CONSTRAINT chk_monitor_type
    CHECK (type IN ('ping', 'http', 'tcp', 'dns'));

-- Constraint for valid status values
ALTER TABLE monitors ADD CONSTRAINT chk_monitor_status
    CHECK (status IN ('unknown', 'up', 'down', 'degraded'));
