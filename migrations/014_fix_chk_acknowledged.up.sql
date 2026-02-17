-- Allow incidents to be auto-resolved (open -> resolved) without acknowledgement.
-- Previously the constraint required acknowledged_by/acknowledged_at for resolved status,
-- which blocked automatic resolution when a monitor recovers.
ALTER TABLE incidents DROP CONSTRAINT chk_acknowledged;
ALTER TABLE incidents ADD CONSTRAINT chk_acknowledged
    CHECK (
        status = 'open'
        OR (status = 'acknowledged' AND acknowledged_by IS NOT NULL AND acknowledged_at IS NOT NULL)
        OR status = 'resolved'
    );
