CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL DEFAULT 'default',
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    current_step INT NOT NULL DEFAULT 0,
    input JSONB,
    output JSONB,
    error TEXT,
    max_retries INT NOT NULL DEFAULT 3,
    retry_count INT NOT NULL DEFAULT 0,
    timeout_at TIMESTAMPTZ,
    locked_by VARCHAR(255),
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflows_status_pending ON workflows (status)
    WHERE status IN ('pending', 'running');

CREATE INDEX idx_workflows_tenant_created ON workflows (tenant_id, created_at DESC);
