```
██╗    ██╗ █████╗ ████████╗ ██████╗██╗  ██╗██████╗  ██████╗  ██████╗
██║    ██║██╔══██╗╚══██╔══╝██╔════╝██║  ██║██╔══██╗██╔═══██╗██╔════╝
██║ █╗ ██║███████║   ██║   ██║     ███████║██║  ██║██║   ██║██║  ███╗
██║███╗██║██╔══██║   ██║   ██║     ██╔══██║██║  ██║██║   ██║██║   ██║
╚███╔███╔╝██║  ██║   ██║   ╚██████╗██║  ██║██████╔╝╚██████╔╝╚██████╔╝
 ╚══╝╚══╝ ╚═╝  ╚═╝   ╚═╝    ╚═════╝╚═╝  ╚═╝╚═════╝  ╚═════╝  ╚═════╝
```
**Infrastructure Monitoring for Hybrid Environments**

![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)
![Echo](https://img.shields.io/badge/Echo-v4-00ADD8)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![TimescaleDB](https://img.shields.io/badge/TimescaleDB-Hypertable-FDB515)
![HTMX](https://img.shields.io/badge/HTMX-Real--Time-3366CC)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)

[Quick Start](#quick-start) • [Configuration](#configuration) • [Deployment](#deployment) • [Documentation](#documentation)

---

## What is WatchDog?

WatchDog is a monitoring system that uses a **Private Agent** architecture to monitor internal labs, databases behind firewalls, and public APIs with equal ease. Agents run inside your network and connect **outbound** to the Hub — no inbound firewall rules required.

**Key Features:**

- **Private Agent Architecture** — Monitor internal resources without opening inbound firewall ports
- **Real-Time Dashboard** — Live status updates via SSE and HTMX, no page refresh needed
- **3-Strike Rule** — Verifies failures before alerting, eliminating false positives from transient issues
- **Incident Lifecycle** — Automatic creation, acknowledgment workflow, and resolution with TTR metrics
- **Zero-Config Agent** — Agents need only an API key, all configuration is pushed from the Hub
- **Multi-Channel Alerts** — Pluggable notification system with Slack, Discord, and webhook support
- **Tier-Based Plans** — Free, Pro, and Team plans with configurable limits

## Architecture

```
+-----------------------------------------------------------------+
|                            CLOUD                                |
|  +-----------------------------------------------------------+  |
|  |                      HUB SERVER                           |  |
|  |  +----------+  +-----------+  +------------------------+  |  |
|  |  | REST API |  | WebSocket |  |      Dashboard         |  |  |
|  |  +----------+  +-----------+  | (Go Templates + HTMX)  |  |  |
|  |                               +------------------------+  |  |
|  |  +-----------------------------------------------------+  |  |
|  |  |          PostgreSQL + TimescaleDB                   |  |  |
|  |  +-----------------------------------------------------+  |  |
|  +-----------------------------------------------------------+  |
+-----------------------------------------------------------------+
                               ^
                               | WebSocket (outbound only)
                               |
+-----------------------------------------------------------------+
|                    CUSTOMER NETWORK                             |
|  +-----------------------------------------------------------+  |
|  |                       AGENT                               |  |
|  |  +---------+  +---------+  +---------+  +---------+       |  |
|  |  |  HTTP   |  |   TCP   |  |  Ping   |  |   DNS   |       |  |
|  |  | Checker |  | Checker |  | Checker |  | Checker |       |  |
|  |  +---------+  +---------+  +---------+  +---------+       |  |
|  +-----------------------------------------------------------+  |
|                    |              |              |              |
|                    v              v              v              |
|              [Database]    [Internal API]   [Service]           |
+-----------------------------------------------------------------+
```

The system is split across three repositories:

| Repository | Description |
|------------|-------------|
| [watchdog](https://github.com/sylvester-francis/watchdog) (this repo) | Hub server — dashboard, API, alerting, data storage |
| [watchdog-agent](https://github.com/sylvester-francis/watchdog-agent) | Lightweight monitoring agent binary |
| [watchdog-proto](https://github.com/sylvester-francis/watchdog-proto) | Shared WebSocket message protocol |

## Quick Start

**Prerequisites:** Go 1.23+, Docker, Make

```bash
# Clone the repository
git clone https://github.com/sylvester-francis/watchdog.git
cd watchdog

# Install development tools
make install-tools

# Start the database
make docker-db

# Run database migrations
make migrate-up

# Start the Hub (with hot reload)
make dev-hub
```

The Hub will be available at `http://localhost:8080`.

### Connect an Agent

1. Register a user account through the dashboard
2. Create an agent in the dashboard (generates an API key)
3. Install and run the agent:

```bash
curl -sSL https://raw.githubusercontent.com/sylvester-francis/watchdog-agent/main/scripts/install-agent.sh | sudo sh -s -- \
  --api-key YOUR_API_KEY \
  --hub-url wss://your-hub.example.com/ws/agent
```

See the [watchdog-agent README](https://github.com/sylvester-francis/watchdog-agent) for detailed installation options.

## Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | — | Yes |
| `ENCRYPTION_KEY` | 32-byte hex key for AES-256 API key encryption | — | Yes |
| `SESSION_SECRET` | Session signing key (minimum 32 bytes) | — | Yes |
| `SERVER_HOST` | Server bind address | `0.0.0.0` | No |
| `SERVER_PORT` | Server port | `8080` | No |
| `SECURE_COOKIES` | Set session cookies with Secure flag | `false` | No |

## Deployment

### Docker

```bash
# Build and start the full stack (Hub + PostgreSQL + TimescaleDB)
make docker-build
make docker-up

# View logs
make docker-logs

# Stop
make docker-down
```

### Build from Source

```bash
make build-hub
# Output: bin/hub
```

### Database Migrations

```bash
make migrate-up       # Apply all pending migrations
make migrate-down     # Rollback the last migration
make migrate-create NAME=add_new_table
```

## Development

```bash
make dev-hub          # Hot reload via Air
make test             # Run tests with race detection
make test-short       # Skip slow tests
make test-coverage    # HTML coverage report
make lint             # golangci-lint
make fmt              # gofmt + goimports
make sec              # gosec security scan
make vuln             # govulncheck
```

## Plan Limits

| Plan | Max Agents | Max Monitors |
|------|------------|--------------|
| Free | 1 | 3 |
| Pro | 3 | 25 |
| Team | 10 | Unlimited |

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](docs/ARCHITECTURE.md) | Project structure, hexagonal architecture, design principles |
| [API Reference](docs/API.md) | REST endpoints, WebSocket protocol, response formats |
| [Database Schema](docs/SCHEMA.md) | Tables, migrations, TimescaleDB hypertables |
| [Security](docs/SECURITY.md) | Password hashing, encryption, rate limiting, input validation |

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go |
| Web Framework | Echo v4 |
| Database | PostgreSQL 16 + TimescaleDB |
| Frontend | Go Templates + HTMX + TailwindCSS |
| Real-Time | WebSockets (agents) + SSE (dashboard) |
| Auth | Argon2id + AES-GCM + gorilla/sessions |
| Deployment | Docker + Docker Compose + Railway |

## Related Repositories

| Repository | Description |
|------------|-------------|
| [watchdog-agent](https://github.com/sylvester-francis/watchdog-agent) | Lightweight monitoring agent binary |
| [watchdog-proto](https://github.com/sylvester-francis/watchdog-proto) | Shared WebSocket message protocol |

## License

This software is proprietary. All rights reserved.
