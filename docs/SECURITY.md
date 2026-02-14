# Security

## Password Hashing

User passwords are hashed with **Argon2id** using OWASP-recommended parameters:

| Parameter | Value |
|-----------|-------|
| Memory | 64 MB |
| Iterations | 3 |
| Threads | 4 |
| Salt | 16 bytes (cryptographically random) |
| Key Length | 32 bytes |

Verification uses constant-time comparison to prevent timing attacks.

## API Key Encryption

Agent API keys are encrypted at rest using **AES-256-GCM**:

- Keys are generated as cryptographically random 32-byte values
- Stored encrypted in the `agents.api_key_encrypted` column (BYTEA)
- Decrypted only during validation
- The encryption key is configured via the `ENCRYPTION_KEY` environment variable (32-byte hex string)

### API Key Format

API keys follow the format `agentID:secret` for O(1) lookup by agent ID during WebSocket authentication.

## Session Security

Sessions are managed with `gorilla/sessions` using secure cookies:

| Setting | Value |
|---------|-------|
| MaxAge | 7 days |
| HttpOnly | true |
| SameSite | Lax |
| Secure | Configurable via `SECURE_COOKIES` env var |

## Rate Limiting

Token bucket rate limiting is applied to sensitive endpoints:

| Endpoint | Rate | Burst |
|----------|------|-------|
| `/login` | 5 req/min | 10 |
| `/register` | 5 req/min | 10 |
| `/waitlist` | 5 req/min | 10 |

A cleanup goroutine runs periodically to prevent memory leaks from expired buckets.

## Input Validation

All user-supplied input is validated at the boundary:

| Field | Constraint |
|-------|------------|
| Agent name | Max 255 characters |
| Monitor name | Max 255 characters |
| Monitor target | Max 500 characters |
| Email | Valid email format, unique |

## XSS Prevention

All user-supplied input rendered in templates is escaped with `html.EscapeString()`.

## SQL Injection Prevention

All database queries use parameterized queries via `pgx`. No string concatenation is used for query building.

## Secure Headers

The secure headers middleware applies the following headers to all responses:

| Header | Value |
|--------|-------|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Content-Security-Policy` | Restrictive policy |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |

## WebSocket Authentication

- Agents must authenticate within 10 seconds of connecting
- API key validation uses constant-time comparison
- Unauthenticated connections are terminated
- Each agent connection is tracked in the Hub's client registry
