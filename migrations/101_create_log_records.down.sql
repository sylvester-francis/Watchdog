SELECT remove_retention_policy('log_records', if_exists => true);
SELECT remove_compression_policy('log_records', if_exists => true);

DELETE FROM system_settings WHERE key = 'log_retention_days';

DROP TABLE IF EXISTS log_records;
