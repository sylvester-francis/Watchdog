# Architecture

## Project Structure

```
watchdog/
    cmd/
        hub/                    # Hub server entrypoint
    internal/
        core/
            domain/             # Business entities (User, Agent, Monitor, Incident, Heartbeat)
            ports/              # Interfaces (repositories, services)
            services/           # Business logic (auth, monitor, incident)
            realtime/           # WebSocket Hub and Client
        adapters/
            http/
                handlers/       # Echo HTTP handlers
                middleware/     # Auth, rate limiting, secure headers
                view/           # Template renderer
            repository/         # PostgreSQL implementations
            notify/             # Alert adapters (Slack, Discord, webhook)
        crypto/                 # Argon2id password hashing, AES-GCM encryption
        config/                 # Environment-based configuration
    web/
        templates/              # Go HTML templates (layouts, pages, partials)
        static/                 # CSS and JavaScript assets
    migrations/                 # SQL migrations (golang-migrate)
    deployments/
        Dockerfile.hub          # Multi-stage Docker build
        docker-compose.yml      # Full stack (Hub + PostgreSQL + TimescaleDB)
    scripts/
        railway-migrate.sh      # Railway deployment migration entrypoint
```

## Hexagonal Architecture

The codebase follows **Hexagonal Architecture** (Ports and Adapters):

```
            +------------------+
            |     Adapters     |
            |  (HTTP, Repo,    |
            |   Notify)        |
            +--------+---------+
                     |
                     v
            +------------------+
            |      Ports       |
            |  (Interfaces)    |
            +--------+---------+
                     |
                     v
            +------------------+
            |    Services      |
            | (Business Logic) |
            +--------+---------+
                     |
                     v
            +------------------+
            |     Domain       |
            | (Entities/Types) |
            +------------------+
```

- **Domain** — Pure business entities with no external dependencies. Defines `User`, `Agent`, `Monitor`, `Incident`, `Heartbeat`, and plan types.
- **Ports** — Interfaces that define what the application needs. Repository contracts (`UserRepository`, `MonitorRepository`, etc.) and service contracts (`UserAuthService`, `MonitorService`, etc.).
- **Services** — Business logic that depends only on port interfaces. Auth service handles registration/login, monitor service orchestrates heartbeats and the 3-strike rule, incident service manages the state machine.
- **Adapters** — Concrete implementations. PostgreSQL repositories, Echo HTTP handlers, Slack/Discord/webhook notifiers.

## Design Principles

These are non-negotiable rules enforced throughout the codebase:

1. **Trust No One** — The Hub verifies the agent API key on every WebSocket handshake. Keys are encrypted at rest with AES-GCM.
2. **Verify Before Alerting** — Never alert on first failure. A monitor must fail three consecutive checks before an incident is opened.
3. **Data Integrity** — Database transactions are used when updating monitor status and creating incidents simultaneously.
4. **Zero-Config Agent** — The agent requires only an API key. All monitoring configuration is pushed from the Hub after authentication.

## Incident State Machine

```
             +-------------------------------------+
             |                                     |
             v                                     |
+--------+  3 failures  +--------+  recover  +----------+
| Normal |------------->|  Open  |---------->| Resolved |
+--------+              +--------+           +----------+
                            |
                       acknowledge
                            |
                            v
                      +----------+
                      |  Acked   |
                      +----------+
```

- Incidents are created automatically after 3 consecutive heartbeat failures for a monitor.
- Users can acknowledge open incidents to signal awareness.
- Incidents are resolved automatically when the monitor recovers, or manually by a user.
- Time-to-resolve (TTR) is calculated on resolution.

## WebSocket Hub

The real-time layer uses a central Hub pattern:

- **Hub** — Central event loop that manages agent connections. Thread-safe client registry with `RWMutex`. Supports register, unregister, broadcast, and per-agent message routing.
- **Client** — Per-agent WebSocket connection handler. Manages read/write pumps with ping/pong keepalive.
- **Protocol** — JSON messages defined in [watchdog-proto](https://github.com/sylvester-francis/watchdog-proto). Auth handshake, task assignment, heartbeat reporting.

## Real-Time Dashboard

The dashboard uses **Server-Sent Events (SSE)** to push updates to the browser without polling:

1. Agent sends heartbeat via WebSocket to the Hub
2. Hub processes the heartbeat (updates DB, checks 3-strike rule)
3. Hub broadcasts status change via SSE to connected dashboard clients
4. HTMX swaps the updated HTML fragment into the DOM

## Transaction Management

The codebase uses a context-based `Transactor` pattern for database transactions:

- Services call `transactor.WithinTransaction(ctx, fn)` to wrap multi-step operations
- The transaction-aware context is passed through the call chain
- Repositories check for an active transaction in the context and use it if present
- If any step fails, the entire transaction is rolled back
