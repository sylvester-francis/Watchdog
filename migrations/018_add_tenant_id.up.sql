-- Add tenant_id to all tenant-scoped tables
ALTER TABLE users ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE agents ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE monitors ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE heartbeats ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE incidents ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE status_pages ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE alert_channels ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE api_tokens ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE audit_logs ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE usage_events ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';

-- Indexes for tenant filtering
CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_agents_tenant ON agents(tenant_id);
CREATE INDEX idx_monitors_tenant ON monitors(tenant_id);
CREATE INDEX idx_heartbeats_tenant ON heartbeats(tenant_id, time DESC);
CREATE INDEX idx_incidents_tenant ON incidents(tenant_id);
CREATE INDEX idx_status_pages_tenant ON status_pages(tenant_id);
CREATE INDEX idx_alert_channels_tenant ON alert_channels(tenant_id);
CREATE INDEX idx_api_tokens_tenant ON api_tokens(tenant_id);
CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_usage_events_tenant ON usage_events(tenant_id);

-- Update unique constraints to be tenant-scoped
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
ALTER TABLE users ADD CONSTRAINT users_tenant_email_key UNIQUE(tenant_id, email);
DROP INDEX IF EXISTS idx_users_email;
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;
ALTER TABLE users ADD CONSTRAINT users_tenant_username_key UNIQUE(tenant_id, username);
