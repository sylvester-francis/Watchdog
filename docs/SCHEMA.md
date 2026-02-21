# Database Schema

## Tables

### users

User accounts with Argon2id password hashes.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    plan VARCHAR(20) DEFAULT 'free',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### agents

Registered monitoring agents with AES-GCM encrypted API keys.

```sql
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    api_key_encrypted BYTEA NOT NULL,
    last_seen_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'offline',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### monitors

Monitoring targets with configurable check type, interval, and timeout.

```sql
CREATE TABLE monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,         -- http, tcp, ping, dns
    target VARCHAR(500) NOT NULL,
    interval_seconds INT DEFAULT 30,
    timeout_seconds INT DEFAULT 10,
    status VARCHAR(20) DEFAULT 'unknown',
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### heartbeats

TimescaleDB hypertable for time-series check results. Automatically partitioned by time.

```sql
CREATE TABLE heartbeats (
    time TIMESTAMPTZ NOT NULL,
    monitor_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,       -- up, down, timeout, error
    latency_ms INT,
    error_message TEXT,
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);
SELECT create_hypertable('heartbeats', 'time');
```

### incidents

Incident lifecycle with acknowledgment tracking and time-to-resolve.

```sql
CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID REFERENCES monitors(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    ttr_seconds INT,
    acknowledged_by UUID REFERENCES users(id),
    acknowledged_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'open', -- open, acknowledged, resolved
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### usage_events

Plan limit tracking for analytics and approaching-limit notifications.

### waitlist_signups

Beta waitlist email collection.

## Migrations

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate) and stored in `migrations/`.

| Migration | Description |
|-----------|-------------|
| 001 | Create users table |
| 002 | Create agents table |
| 003 | Create monitors table |
| 004 | Create heartbeats hypertable |
| 005 | Create incidents table |
| 006 | Add user plan column |
| 007 | Create waitlist table |

### Commands

```bash
make migrate-up                        # Apply all pending migrations
make migrate-down                      # Rollback the last migration
make migrate-create NAME=add_new_table # Create a new migration pair
```
