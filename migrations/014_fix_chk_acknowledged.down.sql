-- Revert to strict constraint requiring acknowledged fields for resolved status.
ALTER TABLE incidents DROP CONSTRAINT chk_acknowledged;
ALTER TABLE incidents ADD CONSTRAINT chk_acknowledged
    CHECK (
        (status IN ('acknowledged', 'resolved') AND acknowledged_by IS NOT NULL AND acknowledged_at IS NOT NULL)
        OR status = 'open'
    );
