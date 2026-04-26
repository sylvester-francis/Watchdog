SELECT remove_retention_policy('spans', if_exists => true);
SELECT remove_compression_policy('spans', if_exists => true);

DROP TABLE IF EXISTS spans;
DROP TABLE IF EXISTS system_settings;
