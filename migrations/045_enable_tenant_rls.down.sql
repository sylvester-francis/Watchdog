-- Rollback migration 045: Remove tenant RLS policies and disable RLS.

DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN
        SELECT unnest(ARRAY[
            'users',
            'agents',
            'monitors',
            'heartbeats',
            'incidents',
            'status_pages',
            'alert_channels',
            'api_tokens',
            'workflows',
            'maintenance_windows',
            'cert_details'
        ])
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', tbl);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', tbl);
    END LOOP;
END $$;

-- Remove tenant isolation from audit_logs but keep tamper protection RLS.
DROP POLICY IF EXISTS tenant_isolation ON audit_logs;
-- Note: audit_logs RLS remains ENABLED due to migration 030 (tamper protection).
