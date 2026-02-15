# API Reference

## REST Endpoints

### Public Routes

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/` | Landing page |
| `GET` | `/login` | Login form |
| `POST` | `/login` | Login submission (rate-limited) |
| `GET` | `/register` | Register form |
| `POST` | `/register` | Registration submission (rate-limited) |
| `POST` | `/logout` | Logout |
| `POST` | `/waitlist` | Join beta waitlist (rate-limited) |
| `GET` | `/ws/agent` | WebSocket endpoint for agent connections |

### Protected Routes (authenticated users)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/dashboard` | Main dashboard |
| `GET` | `/api/agents` | List agents (JSON) |
| `GET` | `/api/agents/:id` | Get agent details (JSON) |
| `POST` | `/agents` | Create agent |
| `DELETE` | `/agents/:id` | Delete agent |
| `GET` | `/monitors` | List monitors |
| `GET` | `/monitors/new` | Monitor creation form |
| `POST` | `/monitors` | Create monitor |
| `GET` | `/monitors/:id` | Monitor details |
| `GET` | `/monitors/:id/edit` | Monitor edit form |
| `POST` | `/monitors/:id` | Update monitor |
| `DELETE` | `/monitors/:id` | Delete monitor |
| `GET` | `/incidents` | List incidents |
| `GET` | `/incidents/:id` | Incident details |
| `POST` | `/incidents/:id/ack` | Acknowledge incident |
| `POST` | `/incidents/:id/resolve` | Resolve incident |
| `GET` | `/sse/events` | SSE stream for real-time updates |

### Admin Routes

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin` | Admin dashboard with usage statistics |

## Response Format

### Success Response

```json
{
  "data": {},
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100
  }
}
```

### Error Response

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid email format",
    "details": [
      {"field": "email", "message": "must be a valid email"}
    ]
  }
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| `200 OK` | Successful GET/PUT/PATCH |
| `201 Created` | Successful POST |
| `204 No Content` | Successful DELETE |
| `400 Bad Request` | Validation error |
| `401 Unauthorized` | Missing/invalid auth |
| `403 Forbidden` | Valid auth, insufficient permissions |
| `404 Not Found` | Resource doesn't exist |
| `409 Conflict` | Resource conflict (duplicate email) |
| `422 Unprocessable Entity` | Semantic validation error |
| `500 Internal Server Error` | Server-side error |

## WebSocket Protocol

Agents communicate with the Hub using JSON messages defined in [watchdog-proto](https://github.com/sylvester-francis/watchdog-proto).

### Message Envelope

```json
{
  "type": "heartbeat",
  "payload": { ... },
  "timestamp": "2025-01-15T10:30:00Z"
}
```

### Message Types

| Type | Direction | Description |
|------|-----------|-------------|
| `auth` | Agent -> Hub | Agent sends API key to authenticate |
| `auth_ack` | Hub -> Agent | Hub confirms successful authentication |
| `auth_error` | Hub -> Agent | Hub rejects authentication |
| `task` | Hub -> Agent | Hub assigns a monitoring task |
| `heartbeat` | Agent -> Hub | Agent reports check results |
| `ping` | Hub -> Agent | Hub checks agent liveness |
| `pong` | Agent -> Hub | Agent responds to ping |
| `error` | Either | Generic error message |

### Connection Lifecycle

```
Agent                                Hub
  |                                   |
  |-- auth {api_key, version} ------->|
  |                                   |  (validate API key)
  |<------ auth_ack {agent_id} -------|
  |                                   |
  |<------ task {monitor, target} ----|  (one per enabled monitor)
  |<------ task {monitor, target} ----|
  |                                   |
  |-- heartbeat {status, latency} --->|  (after each check)
  |-- heartbeat {status, latency} --->|
  |                                   |
  |<------ ping ----------------------|  (periodic liveness check)
  |-- pong --------------------------->|
```

### Example Messages

**Authentication:**

```json
{"type": "auth", "payload": {"api_key": "agent-uuid:secret-key", "version": "1.0.0"}, "timestamp": "2025-01-15T10:30:00Z"}
```

**Auth Acknowledgment:**

```json
{"type": "auth_ack", "payload": {"agent_id": "uuid", "agent_name": "prod-agent-01"}, "timestamp": "2025-01-15T10:30:00Z"}
```

**Task Assignment:**

```json
{"type": "task", "payload": {"monitor_id": "uuid", "type": "http", "target": "https://api.example.com/health", "interval": 30, "timeout": 10}, "timestamp": "2025-01-15T10:30:00Z"}
```

**Heartbeat:**

```json
{"type": "heartbeat", "payload": {"monitor_id": "uuid", "status": "up", "latency_ms": 42}, "timestamp": "2025-01-15T10:30:01Z"}
```
