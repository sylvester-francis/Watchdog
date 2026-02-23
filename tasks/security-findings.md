# Security Findings

## Resolved

### [CRITICAL] 2026-02-22 — Path traversal in SvelteKit SPA route handler
- **File:** internal/adapters/http/router.go:registerSvelteRoutes()
- **Vector:** User-supplied path param `*` joined to build directory without sanitization, allowing `../` escape
- **Fix:** Added `filepath.Clean()`, `strings.Contains("..")` check, and `strings.HasPrefix` boundary verification
- **Status:** FIXED

### [HIGH] 2026-02-22 — Missing email format validation on JSON auth endpoints
- **File:** internal/adapters/http/handlers/auth_api_handler.go
- **Vector:** Login/register/setup only checked empty string, allowing malformed emails
- **Fix:** Added `net/mail.ParseAddress()` validation + 254-char max length on all three endpoints
- **Status:** FIXED

### [MEDIUM] 2026-02-22 — Token scope not enforced on API endpoints
- **File:** internal/adapters/http/middleware/api_token_auth.go
- **Vector:** `token_scope` is set in context but no handler checks it; read_only tokens can perform writes
- **Fix:** Added `RequireWriteScope()` middleware on v1 group — rejects POST/PUT/DELETE from `read_only` tokens with 403
- **Status:** FIXED

### [MEDIUM] 2026-02-22 — LoginLimiter reads email from FormValue, not JSON body
- **File:** internal/adapters/http/middleware/login_limiter.go
- **Vector:** JSON login requests bypass email-based rate limiting (IP-based limiting still applies via authRL)
- **Fix:** Added `MiddlewareJSON()` method that reads email from JSON body, restores body for downstream; wired on `/api/v1/auth/login`
- **Status:** FIXED

## Open

(none)
