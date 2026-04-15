# WatchDog Webhooks

Outbound webhook notifications for incidents and agent state changes. Optionally signed with HMAC-SHA256 for integrity verification and replay protection.

## Payload format

Two payload shapes — incidents and agent events. Content-Type is always `application/json`.

### Incident events

```json
{
  "event": "incident.opened",
  "timestamp": "2026-04-15T12:34:56Z",
  "incident": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "monitor_id": "660e8400-e29b-41d4-a716-446655440001",
    "status": "open",
    "started_at": "2026-04-15T12:34:56Z",
    "resolved_at": null
  },
  "monitor": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "Production API",
    "type": "http",
    "target": "https://api.example.com/health"
  },
  "context": {
    "error_message": "connection refused",
    "agent_name": "agent-prod-1",
    "interval": "60s",
    "threshold": 3
  }
}
```

Event types: `incident.opened`, `incident.resolved`. For resolved events, `resolved_at` is non-null.

### Agent events

```json
{
  "event_type": "agent.offline",
  "timestamp": "2026-04-15T12:34:56Z",
  "agent_id": "770e8400-e29b-41d4-a716-446655440002",
  "agent_name": "agent-prod-1",
  "affected_monitors": 3
}
```

Event types: `agent.offline`, `agent.online`, `agent.maintenance`. Fields vary by event type.

## Signing (optional)

When a signing secret is configured on the webhook channel, every request includes three headers:

| Header | Value |
|---|---|
| `X-Watchdog-Signature-256` | `sha256=<hex>` — HMAC-SHA256 of the signed string |
| `X-Watchdog-Timestamp` | Unix seconds at send time |
| `X-Watchdog-Nonce` | UUIDv4, fresh per request |

### Signed string construction

```
{timestamp}.{nonce}.{body}
```

Concatenated with literal `.` separators. `body` is the raw JSON payload bytes.

### Algorithm

- HMAC-SHA256
- Key: the configured signing secret (treated as UTF-8 bytes)
- Output: lowercase hex, 64 characters

## Verification recipe

### Go

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "net/http"
    "strconv"
    "strings"
    "time"
)

const webhookSecret = "your-secret"
const replayWindow = 5 * time.Minute

var seenNonces = map[string]bool{} // production: use a TTL-bounded store (Redis, etc.)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "read body", http.StatusBadRequest)
        return
    }

    sigHeader := r.Header.Get("X-Watchdog-Signature-256")
    tsHeader := r.Header.Get("X-Watchdog-Timestamp")
    nonce := r.Header.Get("X-Watchdog-Nonce")

    if sigHeader == "" || tsHeader == "" || nonce == "" {
        http.Error(w, "missing signature headers", http.StatusUnauthorized)
        return
    }

    // Freshness check (replay protection window)
    tsInt, err := strconv.ParseInt(tsHeader, 10, 64)
    if err != nil {
        http.Error(w, "bad timestamp", http.StatusBadRequest)
        return
    }
    if time.Since(time.Unix(tsInt, 0)) > replayWindow {
        http.Error(w, "timestamp outside replay window", http.StatusUnauthorized)
        return
    }

    // Nonce dedup (replay protection)
    if seenNonces[nonce] {
        http.Error(w, "nonce reuse", http.StatusUnauthorized)
        return
    }
    seenNonces[nonce] = true // TTL it in production

    // Signature verification
    provided := strings.TrimPrefix(sigHeader, "sha256=")
    mac := hmac.New(sha256.New, []byte(webhookSecret))
    mac.Write([]byte(tsHeader))
    mac.Write([]byte("."))
    mac.Write([]byte(nonce))
    mac.Write([]byte("."))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))

    if !hmac.Equal([]byte(provided), []byte(expected)) {
        http.Error(w, "signature mismatch", http.StatusUnauthorized)
        return
    }

    // ... process the valid payload ...
    w.WriteHeader(http.StatusOK)
}
```

### Python

```python
import hmac, hashlib, time
from flask import Flask, request, abort

WEBHOOK_SECRET = b"your-secret"
REPLAY_WINDOW = 300  # seconds
seen_nonces = set()  # production: use a TTL cache (redis, memcached)

app = Flask(__name__)

@app.post("/webhook")
def webhook():
    sig = request.headers.get("X-Watchdog-Signature-256", "")
    ts = request.headers.get("X-Watchdog-Timestamp", "")
    nonce = request.headers.get("X-Watchdog-Nonce", "")
    body = request.get_data()

    if not sig or not ts or not nonce:
        abort(401, "missing signature headers")

    if abs(time.time() - int(ts)) > REPLAY_WINDOW:
        abort(401, "timestamp outside replay window")

    if nonce in seen_nonces:
        abort(401, "nonce reuse")
    seen_nonces.add(nonce)

    provided = sig.removeprefix("sha256=")
    expected = hmac.new(
        WEBHOOK_SECRET,
        f"{ts}.{nonce}.".encode() + body,
        hashlib.sha256,
    ).hexdigest()

    if not hmac.compare_digest(provided, expected):
        abort(401, "signature mismatch")

    # ... process the valid payload ...
    return "", 200
```

### Node.js

```javascript
import crypto from "node:crypto";
import express from "express";

const WEBHOOK_SECRET = "your-secret";
const REPLAY_WINDOW_MS = 5 * 60 * 1000;
const seenNonces = new Set(); // production: use a TTL cache

const app = express();
app.use(express.raw({ type: "application/json" })); // preserve raw body

app.post("/webhook", (req, res) => {
    const sig = req.get("X-Watchdog-Signature-256") || "";
    const ts = req.get("X-Watchdog-Timestamp") || "";
    const nonce = req.get("X-Watchdog-Nonce") || "";
    const body = req.body; // Buffer

    if (!sig || !ts || !nonce) return res.status(401).send("missing signature headers");

    const tsMs = Number(ts) * 1000;
    if (Math.abs(Date.now() - tsMs) > REPLAY_WINDOW_MS) {
        return res.status(401).send("timestamp outside replay window");
    }

    if (seenNonces.has(nonce)) return res.status(401).send("nonce reuse");
    seenNonces.add(nonce);

    const provided = sig.replace(/^sha256=/, "");
    const expected = crypto
        .createHmac("sha256", WEBHOOK_SECRET)
        .update(`${ts}.${nonce}.`)
        .update(body)
        .digest("hex");

    const providedBuf = Buffer.from(provided, "hex");
    const expectedBuf = Buffer.from(expected, "hex");
    if (providedBuf.length !== expectedBuf.length ||
        !crypto.timingSafeEqual(providedBuf, expectedBuf)) {
        return res.status(401).send("signature mismatch");
    }

    // ... process the valid payload ...
    res.status(200).end();
});
```

## Replay protection

Replay protection is enforced by the receiver, not the sender. The sender provides enough context to enable it:

- **Timestamp** — reject requests outside a freshness window (5 minutes recommended). Longer windows increase replay exposure; shorter windows risk false rejection from clock drift.
- **Nonce** — dedup per request. Use a TTL-bounded cache (Redis, in-memory with expiry) sized to the freshness window. After the window, a replayed nonce is already rejected by the timestamp check.

Running both checks (timestamp AND nonce) is defense-in-depth. The timestamp check alone is insufficient because an attacker within the window can replay. The nonce check alone requires permanent storage.

## Unsigned webhooks

When no signing secret is configured, the signature headers are absent. The payload format is otherwise identical. Use HTTPS and origin-IP allowlisting for minimum security; prefer signing for untrusted network paths.

## Generating a secret

From the UI: Settings → Alert Channels → Create/Edit Webhook → Generate.

Generates a 256-bit (32-byte) random value, hex-encoded (64 characters). Equivalent shell command:

```bash
openssl rand -hex 32
```

Store the secret securely on the receiver side. The sender stores it encrypted (AES-GCM).
