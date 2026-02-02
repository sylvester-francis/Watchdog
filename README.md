# WatchDog

[![CI](https://github.com/sylvester/watchdog/actions/workflows/ci.yml/badge.svg)](https://github.com/sylvester/watchdog/actions/workflows/ci.yml)
[![CodeQL](https://github.com/sylvester/watchdog/actions/workflows/codeql.yml/badge.svg)](https://github.com/sylvester/watchdog/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/sylvester/watchdog)](https://goreportcard.com/report/github.com/sylvester/watchdog)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**Infrastructure Assurance for Hybrid Environments**

WatchDog is a hybrid infrastructure monitoring system that uses a Private Agent architecture to monitor internal labs, databases behind firewalls, and public APIs with equal ease. It offers real-time feedback, incident lifecycle management, and verified reliability.

## Architecture

WatchDog uses a Hub-and-Spoke system:

```
┌─────────────────────────────────────────────────────────────┐
│                         CLOUD                                │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    HUB SERVER                        │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │    │
│  │  │ REST API │  │WebSocket │  │    Dashboard     │   │    │
│  │  └──────────┘  └──────────┘  └──────────────────┘   │    │
│  │  ┌──────────────────────────────────────────────┐   │    │
│  │  │     PostgreSQL + TimescaleDB                  │   │    │
│  │  └──────────────────────────────────────────────┘   │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ WebSocket (outbound only)
                              │
┌─────────────────────────────┼───────────────────────────────┐
│              CUSTOMER NETWORK (behind firewall)              │
│  ┌──────────────────────────┴──────────────────────────┐    │
│  │                     AGENT                            │    │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌────────┐  │    │
│  │  │  HTTP   │  │   TCP   │  │  Ping   │  │  DNS   │  │    │
│  │  │ Checker │  │ Checker │  │ Checker │  │Checker │  │    │
│  │  └─────────┘  └─────────┘  └─────────┘  └────────┘  │    │
│  └─────────────────────────────────────────────────────┘    │
│                              │                               │
│              ┌───────────────┼───────────────┐              │
│              ▼               ▼               ▼              │
│         [Database]    [Internal API]    [Service]           │
└─────────────────────────────────────────────────────────────┘
```

- **Hub (Cloud Server)**: The brain - manages users, stores data, serves the UI, and orchestrates alerts
- **Agent (Spoke)**: The hands - a lightweight Go binary running inside customer networks

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23+ |
| Web Framework | Echo v4 |
| Database | PostgreSQL 16 + TimescaleDB |
| Frontend | Go Templates + HTMX + TailwindCSS |
| Real-Time | WebSockets (Agents) + SSE (Dashboard) |
| Security | Argon2id (Auth) + AES-GCM (Encryption) |
| Deployment | Docker + Docker Compose |

## Key Features

- **Private Agent Architecture**: Monitor internal resources without inbound firewall rules
- **Real-Time Dashboard**: Live status updates via Server-Sent Events
- **Incident Lifecycle**: Automatic incident creation with acknowledgment workflow
- **3-Strike Rule**: Verified failures before alerting (no false positives)
- **Multi-Platform Agent**: Single binary for Linux, Windows, and macOS
- **Zero-Config Agent**: All configuration pushed from the Hub

## Quick Start

### Prerequisites

- Go 1.23+
- Docker and Docker Compose
- Make
- (Optional) [pre-commit](https://pre-commit.com/) for git hooks

### Installation

```bash
# Clone the repository
git clone https://github.com/sylvester/watchdog.git
cd watchdog

# Install development tools
make install-tools

# Install pre-commit hooks (optional but recommended)
pip install pre-commit
make pre-commit-install
```

### Development

```bash
# Start database
make docker-db

# Run migrations
make migrate-up

# Run hub in development mode (with hot reload)
make dev-hub

# Run agent (in another terminal)
WATCHDOG_API_KEY=your-api-key make dev-agent
```

### Build

```bash
# Build both hub and agent
make build

# Build hub only
make build-hub

# Build agent for all platforms (Linux, macOS, Windows)
make build-agent
```

### Docker Deployment

```bash
# Build images
make docker-build

# Start full stack
make docker-up

# View logs
make docker-logs

# Stop all containers
make docker-down
```

## Testing

```bash
# Run all tests with race detection
make test

# Run quick tests
make test-short

# Generate coverage report
make test-coverage

# Run mutation tests (requires gremlins)
make test-mutation
```

## Code Quality

```bash
# Run linter
make lint

# Run linter with auto-fix
make lint-fix

# Format code
make fmt

# Run security scan
make sec

# Check for vulnerabilities
make vuln

# Run all pre-commit hooks
make pre-commit-run
```

## Project Structure

```
watchdog/
├── cmd/
│   ├── hub/            # Hub server entrypoint
│   └── agent/          # Agent binary entrypoint
├── internal/
│   ├── core/
│   │   ├── domain/     # Business entities (User, Agent, Monitor, etc.)
│   │   ├── ports/      # Interfaces (Repository, Services)
│   │   ├── services/   # Business logic
│   │   └── realtime/   # WebSocket Hub & Client
│   ├── adapters/
│   │   ├── http/       # Echo handlers + middleware
│   │   ├── repository/ # PostgreSQL implementations
│   │   └── notify/     # Alert adapters (Discord, Slack)
│   ├── crypto/         # Argon2id & AES-GCM utilities
│   └── config/         # Configuration management
├── web/
│   ├── templates/      # Go HTML templates
│   └── static/         # CSS/JS assets
├── migrations/         # SQL migrations (golang-migrate)
├── scripts/            # Build and install scripts
└── deployments/        # Docker Compose & Dockerfiles
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `ENCRYPTION_KEY` | 32-byte key for AES-256 encryption | Required |
| `SESSION_SECRET` | Session signing key (min 32 bytes) | Required |
| `SERVER_HOST` | Server bind address | `0.0.0.0` |
| `SERVER_PORT` | Server port | `8080` |

### Agent Configuration

The agent requires only an API key - all other configuration is pushed from the Hub:

```bash
# Via environment variable
export WATCHDOG_API_KEY=your-api-key
./agent

# Via command line flag
./agent -api-key=your-api-key -hub=ws://hub.example.com/ws/agent
```

## First Principles

> These are **NON-NEGOTIABLE** design principles:

1. **Trust No One**: Server verifies Agent API Key on every handshake
2. **Verify Before Alerting**: Never alert on first failure - use 3-strike backoff
3. **Data Integrity**: Use transactions when updating status and creating incidents
4. **Zero-Config**: Agent requires no config file - all configs pushed from Server

## API Overview

### REST Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/v1/auth/register` | Register new user |
| `POST` | `/api/v1/auth/login` | User login |
| `GET` | `/api/v1/monitors` | List monitors |
| `POST` | `/api/v1/monitors` | Create monitor |
| `GET` | `/api/v1/incidents` | List incidents |
| `POST` | `/api/v1/incidents/:id/ack` | Acknowledge incident |

### WebSocket Protocol

Agents connect via WebSocket and communicate using JSON messages:

```json
{"type": "auth", "payload": {"api_key": "..."}, "timestamp": "..."}
{"type": "task", "payload": {"monitor_id": "...", "type": "http", "target": "..."}}
{"type": "heartbeat", "payload": {"monitor_id": "...", "status": "up", "latency_ms": 42}}
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Install pre-commit hooks (`make pre-commit-install`)
4. Make your changes
5. Run tests (`make test`)
6. Run linter (`make lint`)
7. Commit your changes (`git commit -m 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
