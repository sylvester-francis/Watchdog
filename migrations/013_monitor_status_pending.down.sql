-- Revert 'pending' back to 'unknown'
ALTER TABLE monitors DROP CONSTRAINT chk_monitor_status;
ALTER TABLE monitors ADD CONSTRAINT chk_monitor_status
    CHECK (status IN ('unknown', 'up', 'down', 'degraded'));

UPDATE monitors SET status = 'unknown' WHERE status = 'pending';

ALTER TABLE monitors ALTER COLUMN status SET DEFAULT 'unknown';
