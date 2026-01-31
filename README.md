# WatchDog

**Infrastructure Assurance for Hybrid Environments**

WatchDog is a hybrid infrastructure monitoring system that uses a Private Agent architecture to monitor internal labs, databases behind firewalls, and public APIs with equal ease. It offers real-time feedback, incident lifecycle management, and verified reliability.

## Architecture

WatchDog uses a Hub-and-Spoke system:

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

## Project Structure

```
watchdog/
├── cmd/
│   ├── hub/            # Hub server entrypoint
│   └── agent/          # Agent binary entrypoint
├── internal/
│   ├── core/
│   │   ├── domain/     # Business entities
│   │   ├── ports/      # Interfaces
│   │   ├── services/   # Business logic
│   │   └── realtime/   # WebSocket Hub
│   ├── adapters/
│   │   ├── http/       # Echo handlers + middleware
│   │   ├── repository/ # PostgreSQL implementations
│   │   └── notify/     # Alert adapters
│   ├── crypto/         # Security utilities
│   └── config/         # Configuration
├── web/
│   ├── templates/      # Go HTML templates
│   └── static/         # CSS/JS assets
├── migrations/         # SQL migrations
├── scripts/            # Build scripts
└── deployments/        # Docker files
```

## Quick Start

### Prerequisites

- Go 1.23+
- Docker and Docker Compose
- Make

### Development

```bash
# Start database
make docker-up

# Run migrations
make migrate-up

# Run hub in development mode
make dev-hub

# Run agent (in another terminal)
make dev-agent
```

### Build

```bash
# Build hub binary
make build-hub

# Build agent for all platforms
make build-agent
```

### Docker Deployment

```bash
# Build images
make docker-build

# Start full stack
make docker-up
```

## Key Features

- **Private Agent Architecture**: Monitor internal resources without inbound firewall rules
- **Real-Time Dashboard**: Live status updates via Server-Sent Events
- **Incident Lifecycle**: Automatic incident creation with acknowledgment workflow
- **3-Strike Rule**: Verified failures before alerting (no false positives)
- **Multi-Platform Agent**: Single binary for Linux, Windows, and macOS
- **Zero-Config Agent**: All configuration pushed from the Hub

## First Principles

1. **Trust No One**: Server verifies Agent API Key on every handshake
2. **Verify Before Alerting**: Never alert on first failure - use 3-strike backoff
3. **Data Integrity**: Use transactions when updating status and creating incidents
4. **Zero-Config**: Agent requires no config file - all configs pushed from Server

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
