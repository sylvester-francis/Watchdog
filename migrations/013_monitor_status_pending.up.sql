-- Replace 'unknown' status with 'pending' for monitors
ALTER TABLE monitors DROP CONSTRAINT chk_monitor_status;
ALTER TABLE monitors ADD CONSTRAINT chk_monitor_status
    CHECK (status IN ('pending', 'up', 'down', 'degraded'));

-- Update existing 'unknown' rows
UPDATE monitors SET status = 'pending' WHERE status = 'unknown';

-- Change default
ALTER TABLE monitors ALTER COLUMN status SET DEFAULT 'pending';
