# WatchDog

**Infrastructure assurance for hybrid environments.**

WatchDog is a hub-and-spoke monitoring system that lets you monitor internal labs, databases behind firewalls, and public APIs from a single dashboard. You deploy lightweight agents inside your private networks — they phone home to the hub, run health checks, and report results in real-time. No inbound firewall rules needed.

---

## How It Works

```
                    Internet
                       │
          ┌────────────┴────────────┐
          │        WatchDog Hub     │
          │    (Your Public Server) │
          │                         │
          │  Dashboard ◄── SSE ──── Web Browser
          │  REST API               │
          │  WebSocket Server       │
          │  PostgreSQL/TimescaleDB │
          └──────┬──────────┬───────┘
                 │          │
            WebSocket    WebSocket
            (outbound)   (outbound)
                 │          │
          ┌──────┴───┐ ┌───┴──────┐
          │  Agent A  │ │  Agent B │
          │  Office   │ │  AWS VPC │
          │  Network  │ │  Private │
          └──────┬────┘ └───┬──────┘
                 │          │
           ping/http/tcp  ping/http/tcp
                 │          │
          ┌──────┴───┐ ┌───┴──────┐
          │ Internal  │ │ Database │
          │ Services  │ │ Cluster  │
          └──────────┘ └──────────┘
```

**Agents connect outbound to the Hub via WebSocket.** This means agents work behind NATs, firewalls, and VPNs without any port forwarding. The Hub pushes monitor configurations to agents, and agents push health check results back.

---

## Quick Start

### 1. Start the Hub

```bash
# Start the database
make docker-db

# Apply migrations
make migrate-up

# Set required environment variables
export DATABASE_URL="postgres://watchdog:watchdog@localhost:5432/watchdog?sslmode=disable"
export ENCRYPTION_KEY="your-32-character-encryption-key!"  # exactly 32 bytes
export SESSION_SECRET="your-session-secret-at-least-32-characters-long"

# Run the hub
make dev-hub
```

The hub is now running at `http://localhost:8080`.

### 2. Create an Account

Open `http://localhost:8080/register` and create your account. Then log in.

### 3. Create an Agent

From the dashboard, create a new agent. You'll receive an API key in the format:

```
a1b2c3d4-e5f6-7890-abcd-ef1234567890:64-char-hex-secret
```

**Save this key immediately** — it's shown only once. The format is `agentID:secret`, which allows the Hub to look up the agent in O(1) time and verify the key with constant-time comparison.

### 4. Run the Agent

```bash
# Using flags
./watchdog-agent -hub ws://your-hub:8080/ws/agent -api-key "YOUR_API_KEY"

# Or using environment variable
export WATCHDOG_API_KEY="YOUR_API_KEY"
./watchdog-agent -hub ws://your-hub:8080/ws/agent
```

The agent connects, authenticates, receives its monitor tasks, and begins executing health checks.

### 5. Create Monitors

From the dashboard at `/monitors`, create monitors and assign them to your agent:

| Type | Target Format | What It Does |
|------|---------------|--------------|
| **HTTP** | `https://api.example.com/health` | GET request, expects 2xx/3xx |
| **TCP** | `db.internal:5432` | Opens TCP connection, verifies port is listening |
| **Ping** | `192.168.1.1` | TCP connect to port 80/443 (no raw ICMP needed) |
| **DNS** | `example.com` | DNS resolution check |

Each monitor has configurable **interval** (5s–3600s, default 30s) and **timeout** (1s–60s, default 10s).

---

## The 3-Strike Rule

WatchDog never alerts on the first failure. When a health check fails:

```
Strike 1: Failure recorded. No alert.
Strike 2: Failure recorded. No alert.
Strike 3: Failure recorded. INCIDENT CREATED. Notifications sent.
```

Only **3 consecutive failures** trigger an incident. This prevents false alarms from transient network blips, garbage collection pauses, or brief service restarts.

When a successful check comes in after an incident is open, the incident is **automatically resolved**, and a recovery notification is sent.

---

## Incident Lifecycle

```
                 3 consecutive         user clicks
    Normal ──── failures ────► Open ────── ACK ──────► Acknowledged
                                │                          │
                                │       successful         │
                                ├──── health check ────────┤
                                │                          │
                                ▼                          ▼
                             Resolved ◄─────────────── Resolved
                           (auto or manual)
                           TTR calculated
```

| Status | Meaning |
|--------|---------|
| **Open** | Monitor is down. Nobody has acknowledged it yet. |
| **Acknowledged** | Someone is looking at it. Still alerting-eligible. |
| **Resolved** | Fixed. Time-to-resolve (TTR) is calculated automatically. |

Incident creation and monitor status updates happen inside a **database transaction** — they either both succeed or neither does.

---

## Real-Time Dashboard

The dashboard uses **Server-Sent Events (SSE)** for live updates. When an agent comes online, goes offline, or an incident is created/resolved, the dashboard updates within 5 seconds. No polling, no page refreshes.

The UI is built with:
- **Go HTML templates** — server-rendered, fast, no JavaScript framework
- **HTMX** — dynamic updates without full page reloads
- **Tailwind CSS** — dark theme, responsive

---

## Notifications

WatchDog sends alerts when incidents open and resolve. Three notifier adapters are included:

### Discord

```bash
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
```

Sends rich embeds with monitor name, type, target, and timing.

### Slack

```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
```

Sends attachments with color-coded status (red for down, green for resolved).

### Generic Webhook

```bash
export WEBHOOK_URL="https://your-service.com/webhook"
```

Sends a JSON payload:

```json
{
  "event": "incident.opened",
  "timestamp": "2024-01-15T10:30:00Z",
  "incident": {
    "id": "uuid",
    "monitor_id": "uuid",
    "status": "open",
    "started_at": "2024-01-15T10:30:00Z"
  },
  "monitor": {
    "id": "uuid",
    "name": "Production API",
    "type": "http",
    "target": "https://api.example.com/health"
  }
}
```

Notifiers are composable — use `MultiNotifier` to send to Discord and Slack simultaneously. Use `NoOpNotifier` for testing.

---

## Configuration Reference

### Hub Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `ENCRYPTION_KEY` | Yes | — | 32-byte key for AES-256-GCM encryption of API keys |
| `SESSION_SECRET` | Yes | — | 32+ byte secret for session cookies |
| `SERVER_HOST` | No | `0.0.0.0` | Bind address |
| `SERVER_PORT` | No | `8080` | Listen port |
| `SERVER_READ_TIMEOUT` | No | `15s` | HTTP read timeout |
| `SERVER_WRITE_TIMEOUT` | No | `15s` | HTTP write timeout |
| `DATABASE_MAX_CONNS` | No | `25` | Connection pool max size |
| `DATABASE_MIN_CONNS` | No | `5` | Connection pool min size |

### Agent CLI

```
Usage: watchdog-agent [flags]

Flags:
  -hub string        Hub WebSocket URL (default "ws://localhost:8080/ws/agent")
  -api-key string    Agent API key (or set WATCHDOG_API_KEY)
  -debug             Enable debug logging
  -version           Print version and exit
```

---

## API Reference

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Returns `{"status": "healthy"}` |
| GET | `/ws/agent` | WebSocket endpoint for agents (API key auth) |

### Auth Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/login` | Login page |
| POST | `/login` | Submit login (form: `email`, `password`) |
| GET | `/register` | Registration page |
| POST | `/register` | Submit registration (form: `email`, `password`, `confirm_password`) |
| POST | `/logout` | End session |

### Dashboard (Auth Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/dashboard` | Main dashboard with agent status and incidents |
| GET | `/sse/events` | SSE stream for real-time updates |

### Agents (Auth Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/agents` | List agents (JSON) |
| GET | `/api/agents/:id` | Get agent (JSON) |
| POST | `/agents` | Create agent (form: `name`). Returns API key **once**. |
| DELETE | `/agents/:id` | Delete agent and all its monitors |

### Monitors (Auth Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/monitors` | List all monitors |
| GET | `/monitors/new` | Create monitor form |
| POST | `/monitors` | Create monitor (form: `agent_id`, `name`, `type`, `target`, `interval`, `timeout`) |
| GET | `/monitors/:id` | Monitor detail |
| GET | `/monitors/:id/edit` | Edit monitor form |
| POST | `/monitors/:id` | Update monitor |
| DELETE | `/monitors/:id` | Delete monitor |

### Incidents (Auth Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/incidents` | List active incidents |
| GET | `/incidents/:id` | Incident detail |
| POST | `/incidents/:id/ack` | Acknowledge incident |
| POST | `/incidents/:id/resolve` | Manually resolve incident |

---

## Deployment

### Docker Compose (Recommended)

```bash
# Set your encryption key
export ENCRYPTION_KEY="your-32-character-encryption-key!"

# Start everything
docker compose -f deployments/docker-compose.yml up -d

# Apply migrations
make migrate-up
```

This starts:
- **PostgreSQL 16 + TimescaleDB** — time-series optimized database
- **WatchDog Hub** — the central server

### Build Agent for Distribution

```bash
make build-agent
```

Produces binaries for:
- `bin/agent-linux-amd64`
- `bin/agent-linux-arm64`
- `bin/agent-darwin-amd64`
- `bin/agent-darwin-arm64`
- `bin/agent-windows-amd64.exe`

### Install Agent on a Server

```bash
./scripts/install-agent.sh --api-key "YOUR_KEY" --hub "ws://hub.example.com:8080/ws/agent"
```

On systems with systemd, this creates and starts a `watchdog-agent` service that auto-restarts on failure and survives reboots.

---

## Database

WatchDog uses **PostgreSQL 16** with the **TimescaleDB** extension for time-series data.

### Schema

- **users** — accounts with Argon2id-hashed passwords
- **agents** — deployed monitoring agents with AES-256-GCM encrypted API keys
- **monitors** — check configurations (type, target, interval, timeout)
- **heartbeats** — TimescaleDB hypertable, auto-compressed after 7 days, auto-purged after 90 days
- **incidents** — lifecycle-tracked with TTR calculation

### Migrations

```bash
make migrate-up      # Apply all pending migrations
make migrate-down    # Rollback last migration
```

---

## Security

| Layer | Protection |
|-------|-----------|
| **Passwords** | Argon2id hashing (OWASP recommended parameters) |
| **API Keys** | AES-256-GCM encryption at rest, `agentID:secret` format |
| **Key Comparison** | `crypto/subtle.ConstantTimeCompare` — timing-attack resistant |
| **Sessions** | HttpOnly, SameSite=Lax cookies (7-day expiry) |
| **SQL** | Parameterized queries throughout — no string concatenation |
| **Headers** | Secure headers middleware (CSP, X-Frame-Options, etc.) |
| **Rate Limiting** | Middleware on auth endpoints |
| **Agent Auth** | API key validated on every WebSocket handshake |
| **Data Integrity** | Incident creation + monitor status update wrapped in DB transactions |

### Trust No One

The Hub verifies the agent's API key on **every** WebSocket connection. There are no cached sessions, no trust-on-first-use shortcuts. If an agent's key is compromised, delete the agent from the dashboard — the old key immediately stops working.

---

## Architecture

```
watchdog/
├── cmd/
│   ├── hub/           # Hub server binary
│   └── agent/         # Agent binary
├── internal/
│   ├── core/
│   │   ├── domain/    # Business entities (User, Agent, Monitor, Incident, Heartbeat)
│   │   ├── ports/     # Interfaces (UserAuthService, AgentAuthService, MonitorService, etc.)
│   │   ├── services/  # Business logic (3-strike rule, incident state machine)
│   │   └── realtime/  # WebSocket hub, client, message protocol
│   ├── adapters/
│   │   ├── http/      # Echo handlers, middleware, router
│   │   ├── repository/# PostgreSQL implementations
│   │   └── notify/    # Discord, Slack, Webhook notifiers
│   ├── crypto/        # Argon2id, AES-GCM
│   └── config/        # Environment-based configuration
├── web/               # HTML templates, CSS, JS
├── migrations/        # SQL migration files
├── deployments/       # Dockerfiles, docker-compose
└── scripts/           # Build and install scripts
```

The architecture follows **hexagonal/ports-and-adapters**:

- **Domain** defines the business rules (what an incident is, when to alert)
- **Ports** define interfaces (what a repository must do, what a service must do)
- **Services** implement business logic against interfaces
- **Adapters** implement the interfaces with real infrastructure (PostgreSQL, HTTP, WebSocket)

Dependencies always point inward: adapters depend on ports, services depend on ports, nothing depends on adapters.

---

## Development

```bash
# Install dependencies
make deps

# Start database
make docker-db

# Run migrations
make migrate-up

# Run hub with hot reload
make dev-hub

# Run agent (in another terminal)
make dev-agent

# Run tests
make test

# Run linter
make lint

# Build everything
make build
```

---

## Make Targets

| Target | Description |
|--------|-------------|
| `make dev` | Start DB + hub with hot reload |
| `make dev-hub` | Hub with hot reload (Air) |
| `make dev-agent` | Run agent locally |
| `make build` | Build hub + agent binaries |
| `make build-agent` | Multi-platform agent build |
| `make test` | Run all tests with race detection |
| `make lint` | Run golangci-lint |
| `make migrate-up` | Apply database migrations |
| `make migrate-down` | Rollback last migration |
| `make docker-up` | Start all Docker containers |
| `make docker-down` | Stop all Docker containers |
| `make clean` | Remove build artifacts |
