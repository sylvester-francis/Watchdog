-- Reverse migration 102.
-- Note: this restores the unscoped schema. Re-running 102 will TRUNCATE
-- any data accumulated under the scoped schema, so down-then-up is a
-- destructive operation by design.

DROP INDEX IF EXISTS idx_log_records_user_tenant_ts;
DROP INDEX IF EXISTS idx_spans_user_tenant_start;

ALTER TABLE log_records DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE log_records DROP COLUMN IF EXISTS user_id;

ALTER TABLE spans DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE spans DROP COLUMN IF EXISTS user_id;
