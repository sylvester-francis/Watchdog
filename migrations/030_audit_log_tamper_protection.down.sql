-- H-010: Reverse audit log tamper protection

-- 1. Remove forced RLS for table owner.
ALTER TABLE audit_logs NO FORCE ROW LEVEL SECURITY;

-- 2. Drop all RLS policies.
DROP POLICY IF EXISTS audit_allow_update ON audit_logs;
DROP POLICY IF EXISTS audit_allow_insert ON audit_logs;
DROP POLICY IF EXISTS audit_allow_read ON audit_logs;
DROP POLICY IF EXISTS audit_no_delete ON audit_logs;

-- 3. Disable Row Level Security.
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;

-- 4. Remove the entry_hash column.
ALTER TABLE audit_logs DROP COLUMN IF EXISTS entry_hash;
