ALTER TABLE monitors ADD COLUMN IF NOT EXISTS sla_target_percent DECIMAL(5,2);
