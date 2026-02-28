CREATE TABLE cert_details (
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    last_checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expiry_days INT,
    issuer TEXT,
    sans TEXT[],
    algorithm TEXT,
    key_size INT,
    serial_number TEXT,
    chain_valid BOOLEAN,
    PRIMARY KEY (monitor_id, tenant_id)
);

CREATE INDEX idx_cert_details_tenant ON cert_details(tenant_id);
CREATE INDEX idx_cert_details_expiry ON cert_details(expiry_days) WHERE expiry_days IS NOT NULL;
