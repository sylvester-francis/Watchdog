-- Discovery scans: tracks network discovery operations
CREATE TABLE discovery_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    subnet CIDR NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    host_count INT DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_discovery_scans_user_id ON discovery_scans(user_id);
CREATE INDEX idx_discovery_scans_agent_id ON discovery_scans(agent_id);

-- Discovered devices: individual hosts found during a scan
CREATE TABLE discovered_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id UUID NOT NULL REFERENCES discovery_scans(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip INET NOT NULL,
    hostname TEXT,
    sys_descr TEXT,
    sys_object_id TEXT,
    sys_name TEXT,
    snmp_reachable BOOLEAN DEFAULT false,
    ping_reachable BOOLEAN DEFAULT false,
    suggested_template_id TEXT,
    monitor_created BOOLEAN DEFAULT false,
    discovered_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_discovered_devices_scan_id ON discovered_devices(scan_id);
CREATE INDEX idx_discovered_devices_user_id ON discovered_devices(user_id);
