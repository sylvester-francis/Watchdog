-- Restore original unique constraints
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_tenant_username_key;
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE(username);

DROP INDEX IF EXISTS idx_users_tenant_email;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_tenant_email_key;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE(email);
CREATE INDEX idx_users_email ON users(email);

-- Drop tenant indexes
DROP INDEX IF EXISTS idx_usage_events_tenant;
DROP INDEX IF EXISTS idx_audit_logs_tenant;
DROP INDEX IF EXISTS idx_api_tokens_tenant;
DROP INDEX IF EXISTS idx_alert_channels_tenant;
DROP INDEX IF EXISTS idx_status_pages_tenant;
DROP INDEX IF EXISTS idx_incidents_tenant;
DROP INDEX IF EXISTS idx_heartbeats_tenant;
DROP INDEX IF EXISTS idx_monitors_tenant;
DROP INDEX IF EXISTS idx_agents_tenant;
DROP INDEX IF EXISTS idx_users_tenant;

-- Remove tenant_id columns
ALTER TABLE usage_events DROP COLUMN tenant_id;
ALTER TABLE audit_logs DROP COLUMN tenant_id;
ALTER TABLE api_tokens DROP COLUMN tenant_id;
ALTER TABLE alert_channels DROP COLUMN tenant_id;
ALTER TABLE status_pages DROP COLUMN tenant_id;
ALTER TABLE incidents DROP COLUMN tenant_id;
ALTER TABLE heartbeats DROP COLUMN tenant_id;
ALTER TABLE monitors DROP COLUMN tenant_id;
ALTER TABLE agents DROP COLUMN tenant_id;
ALTER TABLE users DROP COLUMN tenant_id;
