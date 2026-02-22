# Security Implementation Progress

Tracking sheet for all items from `watchdog-security-spec.md`.

## Quick Wins (QW)

| ID | Item | Status | Notes |
|----|------|--------|-------|
| QW-1 | TLS cert expiry data surfaced in heartbeats | Done | Migration 019, cert card on monitor detail page |
| QW-2 | API key scoping (`admin` / `read_only`) + `last_used_ip` tracking | Done | Migration 020, scope selector in settings |
| QW-2b | API key rotation (grace period, dedicated endpoint) | Deferred | Needs design for grace period overlap |
| QW-3 | Agent fingerprinting (hostname, OS, arch, Go version) | Done | Migration 021, stored on first connect, warns on change |
| QW-3b | HMAC message signing | Deferred | Requires shared secret negotiation design |
| QW-3c | TLS enforcement for agent connections | Deferred | Config-driven, needs reverse proxy testing |
| QW-4 | Security headers (CSP, X-Frame-Options, HSTS, Permissions-Policy) | Done | `secure_headers.go` — nonce-based CSP, `'unsafe-inline'` removed from `script-src` |
| QW-5 | Rate limiting (login, general) | Done | `ratelimit.go` |
| QW-6a | Webhook encryption (AES-256-GCM) | Done | `aes_gcm.go` |
| QW-6b | Session cookies (HttpOnly, Secure, SameSiteLax) | Done | Router session config |
| QW-6c | CORS middleware for REST API | Done | Echo CORS on `/api/v1` group |
| QW-6d | Audit logging wired to all handlers | Done | 10 of 12 actions now fire (login handled separately) |
| QW-7 | Landing page security section | Done | Hero badge, 3-card section, comparison table rows |

## Roadmap (R)

| ID | Item | Status | Notes |
|----|------|--------|-------|
| R-1 | Cert expiry auto-alerting (< 14 days) | Deferred | Needs alert channel integration for cert-specific triggers |
| R-2 | Webhook signature verification (HMAC-SHA256) | Deferred | Outbound webhook signing for consumers |
| R-3 | IP allowlisting for API tokens | Deferred | Per-token CIDR list |
| R-4 | Agent binary signing / checksum verification | Deferred | Needs release pipeline integration |
| R-5 | Full audit log viewer in admin UI | Deferred | Data is being collected; needs admin page |
| R-6 | SOC 2 / compliance documentation | Deferred | Process documentation |
