-- Migration 045: Enable Row Level Security on all tenant-scoped tables.
--
-- Defense-in-depth: even if application code forgets a WHERE tenant_id = $N clause,
-- RLS prevents cross-tenant data access. The application must SET LOCAL app.tenant_id
-- at the start of each transaction/query.
--
-- FORCE ensures RLS applies even to the table owner (superuser bypass still works).

-- Helper: create a uniform tenant isolation policy on a table.
-- Policy allows access only when tenant_id matches the session variable app.tenant_id.
-- If app.tenant_id is not set (empty string or missing), deny all access as a fail-closed default.

DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN
        SELECT unnest(ARRAY[
            'users',
            'agents',
            'monitors',
            -- 'heartbeats' excluded: TimescaleDB hypertable with columnstore does not support RLS.
            -- Heartbeats are tenant-scoped via monitor_id FK; application code enforces isolation.
            'incidents',
            'status_pages',
            'alert_channels',
            'api_tokens',
            'workflows',
            'maintenance_windows',
            'cert_details'
        ])
    LOOP
        -- Enable RLS (idempotent).
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', tbl);

        -- Drop existing tenant policy if re-running.
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', tbl);

        -- Create tenant isolation policy.
        -- current_setting('app.tenant_id', true) returns NULL if not set.
        -- We require an exact match — NULL or empty = deny all (fail-closed).
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I
                USING (tenant_id = current_setting(''app.tenant_id'', true))
                WITH CHECK (tenant_id = current_setting(''app.tenant_id'', true))',
            tbl
        );
    END LOOP;
END $$;

-- audit_logs already has RLS (migration 030) for tamper protection.
-- Add tenant isolation policy alongside existing policies.
DROP POLICY IF EXISTS tenant_isolation ON audit_logs;
CREATE POLICY tenant_isolation ON audit_logs
    USING (tenant_id = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
