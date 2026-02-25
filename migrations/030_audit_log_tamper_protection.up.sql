-- H-010: Audit log tamper protection
-- Adds a hash column for integrity verification and RLS policies to prevent deletion.

-- 1. Add entry_hash column for tamper detection (SHA-256 hash of the row contents).
ALTER TABLE audit_logs ADD COLUMN entry_hash VARCHAR(64);

-- 2. Enable Row Level Security on audit_logs.
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- 3. Prevent all DELETE operations via RLS policy.
CREATE POLICY audit_no_delete ON audit_logs FOR DELETE USING (false);

-- 4. Allow SELECT for all roles.
CREATE POLICY audit_allow_read ON audit_logs FOR SELECT USING (true);

-- 5. Allow INSERT for all roles.
CREATE POLICY audit_allow_insert ON audit_logs FOR INSERT WITH CHECK (true);

-- 6. Allow UPDATE for all roles (needed for backfilling entry_hash).
CREATE POLICY audit_allow_update ON audit_logs FOR UPDATE USING (true);

-- 7. Force RLS to apply to the table owner as well.
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;
