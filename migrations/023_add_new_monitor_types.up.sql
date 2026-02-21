ALTER TABLE monitors DROP CONSTRAINT chk_monitor_type;
ALTER TABLE monitors ADD CONSTRAINT chk_monitor_type
    CHECK (type IN ('ping','http','tcp','dns','tls','docker','database','system'));
