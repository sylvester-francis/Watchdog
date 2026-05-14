# Security

This document describes the security properties of the WatchDog hub. For a per-deliverable progress tracker, see `SECURITY-PROGRESS.md`. For webhook signing details, see `webhooks.md`.

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

## API Token Authentication

REST API requests authenticate with Bearer tokens of the form `wd_<32 hex>`. Token lifecycle:

- Generated as cryptographically random 32-byte values
- Only the SHA-256 hash is stored; plaintext is shown once at creation and cannot be recovered
- Tokens are scoped (see table below), track `last_used_at` and `last_used_ip`
- Revocable from **Settings -> API Tokens**

| Scope | What it can do |
|-------|----------------|
| `admin` | Full read/write across `/api/v1` |
| `read_only` | All GET endpoints under `/api/v1` |
| `telemetry_ingest` | Push-only access to `/v1/traces`, `/v1/logs`, `/v1/logs/raw` for OTel collectors and SDKs |

## Agent API Key Encryption

Agent API keys are encrypted at rest using **AES-256-GCM**:

- Keys are generated as cryptographically random values
- Stored encrypted in `agents.api_key_encrypted` (BYTEA)
- Decrypted only during WebSocket authentication
- The data encryption key is configured via `ENCRYPTION_KEY` (32-byte hex string)
- API keys follow the format `agentID:secret` for O(1) lookup during WebSocket handshake

## Session Security

Sessions are managed with `gorilla/sessions` using cookies:

| Setting | Value |
|---------|-------|
| MaxAge | 7 days |
| HttpOnly | true |
| SameSite | Lax |
| Secure | Set via `SECURE_COOKIES` env var (`true` in production behind HTTPS) |

A `NoCacheHeaders` middleware is applied to authenticated routes so that browser back-button after logout cannot reveal cached private pages.

## Rate Limiting

Token-bucket rate limiting protects authentication and registration endpoints (configured in `internal/middleware/ratelimit.go`). Per-IP and per-email login limiting also applies as brute-force protection with progressive lockout. A cleanup goroutine periodically evicts expired buckets.

For exact endpoint coverage and current per-route limits, see the source — limits are tuned over time and a stale list here would be a security footgun. The hub also implements rate-limit-driven IP blocking that surfaces in the admin UI.

## Input Validation

All user-supplied input is validated at the boundary. Representative limits:

| Field | Constraint |
|-------|------------|
| Agent name | Max 255 characters |
| Monitor name | Max 255 characters |
| Monitor target | Max 500 characters |
| Email | Valid email format, unique |

## XSS Prevention

SvelteKit escapes all template-rendered values by default. The codebase does not use `{@html ...}` on user-controlled content. Server-rendered HTML (Go templates) uses `html.EscapeString()` for any user-controlled values.

## SQL Injection Prevention

All database queries use parameterized queries via `pgx`. No string concatenation is used for query building. Repository methods take typed arguments; raw SQL with user values is treated as a code-review red flag.

## Content Security Policy (CSP)

The hub's CSP middleware (`internal/adapters/http/middleware/secure_headers.go`) emits a **per-request cryptographic nonce** (128-bit, `crypto/rand`):

```
default-src 'self';
script-src 'self' 'nonce-<value>' https://cdn.jsdelivr.net;
style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net;
img-src 'self' data: https://validator.swagger.io;
font-src 'self' https://fonts.gstatic.com;
connect-src 'self' https://cdn.jsdelivr.net;
frame-ancestors 'none';
base-uri 'self';
form-action 'self';
object-src 'none';
```

Notes:

- **`script-src`** has no `'unsafe-inline'`, no `'unsafe-eval'`. Inline `<script>` tags in server-rendered HTML carry `nonce="<value>"`. SvelteKit-rendered routes don't need inline scripts. `cdn.jsdelivr.net` is allowed only for Swagger UI on `/docs`. The legacy `unpkg.com` source was removed in the 2026-05-13 hardening pass — no code actually loaded from it.
- **`style-src 'unsafe-inline'`** is retained because SvelteKit transition styles and ~20 dynamic `style=""` attributes (width %, position %) require it. Style injection cannot execute JS, and `connect-src 'self'` blocks data exfiltration via `background-image: url(...)`.
- **`frame-ancestors 'none'`** is the modern equivalent of `X-Frame-Options: DENY`; both are sent for legacy fallback.
- **`base-uri 'self'`** prevents `<base href="evil">` injection from redirecting relative URLs.
- **`form-action 'self'`** prevents `<form action="evil">` exfiltration from XSS or stored content.
- **`object-src 'none'`** blocks `<object>`, `<embed>`, `<applet>`.

## Other Security Headers

| Header | Value |
|--------|-------|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Permissions-Policy` | `geolocation=(), microphone=(), camera=()` |
| `X-Permitted-Cross-Domain-Policies` | `none` |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains; preload` (when `SECURE_COOKIES=true` — set by the operator in production behind HTTPS) |

The deprecated `X-XSS-Protection` header is intentionally **not** set. CSP supersedes it; modern browsers ignore X-XSS-Protection and older browsers had documented XSS-filter side channels that made the header net-negative.

## CSRF Protection

Double-submit cookie pattern via Echo CSRF middleware. All form POSTs include a `_csrf` hidden field. JSON API requests use the `X-CSRF-Token` header. The OTLP receivers under `/v1/*` are exempted from CSRF (Bearer-token-only auth surface; receivers cannot accept browser-origin requests).

## WebSocket Authentication

- Agents must send an `auth` message within 10 seconds of connecting
- API key validation uses constant-time comparison
- Unauthenticated connections are terminated
- Each authenticated agent is tracked in the hub's client registry
- Agent fingerprint (hostname, OS, arch, Go version) is captured on first connect; a fingerprint change emits a warning in the audit log

## Outbound Webhook Signing

Webhook alert channels can be configured with a signing secret. When set, every outbound HTTP POST includes:

| Header | Value |
|---|---|
| `X-Watchdog-Signature-256` | `sha256=<hex>` — HMAC-SHA256 of `{timestamp}.{nonce}.{body}` |
| `X-Watchdog-Timestamp` | Unix seconds at send time (for receiver-side freshness check) |
| `X-Watchdog-Nonce` | UUIDv4 per request (for receiver-side replay dedup) |

See `webhooks.md` for the verification recipe in Go, Python, and Node.js.

## Distributed Tracing and Logs

The OTLP receivers at `/v1/traces`, `/v1/logs`, and `/v1/logs/raw` accept Bearer tokens with the `telemetry_ingest` scope. Tokens are SHA-256 hashed at rest and tracked with `last_used_ip`. Request bodies are size-capped (1 MB compressed) and per-record-capped (64 KB on the decompressed body) to bound zip-bomb exposure. Gzip `Content-Encoding` is accepted and decompressed before parsing.

## Go Toolchain

The hub is built against Go 1.25.10 (current minor at the time of writing). The 1.25.10 bump on 2026-05-13 closed all `govulncheck` reachable stdlib vulnerabilities.

## Reporting Issues

Security issues should be reported privately via a GitHub Security Advisory on this repository.
