-- Remove policies before dropping table
SELECT remove_retention_policy('heartbeats', if_exists => true);
SELECT remove_compression_policy('heartbeats', if_exists => true);

DROP TABLE IF EXISTS heartbeats;
