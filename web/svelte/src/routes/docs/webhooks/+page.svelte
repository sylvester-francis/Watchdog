<script lang="ts">
	import { onMount } from 'svelte';
	import { ShieldCheck, ExternalLink } from 'lucide-svelte';
	import { getAuth } from '$lib/stores/auth.svelte';

	const auth = getAuth();

	onMount(() => {
		auth.check();
	});
</script>

<svelte:head>
	<title>Webhook signing — WatchDog</title>
	<meta name="description" content="WatchDog webhook payloads, HMAC-SHA256 signing, and replay protection. Verification recipes in Go, Python, and Node.js." />
</svelte:head>

<div class="min-h-screen bg-background font-sans text-foreground antialiased">
	<!-- Top bar -->
	<header class="sticky top-0 z-30 border-b border-border bg-background/80 backdrop-blur-sm">
		<div class="mx-auto flex h-14 max-w-3xl items-center px-4 lg:px-8">
			<a href="/" class="flex shrink-0 items-center space-x-2.5">
				<div class="flex h-7 w-7 items-center justify-center rounded-lg bg-accent">
					<ShieldCheck class="h-3.5 w-3.5 text-white" />
				</div>
				<span class="text-sm font-semibold text-foreground">WatchDog</span>
			</a>
			<span class="mx-3 text-border">|</span>
			<span class="text-sm text-muted-foreground">Webhooks</span>

			<div class="ml-auto flex items-center space-x-3">
				<a href="/docs" class="hidden items-center space-x-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground sm:inline-flex">
					<span>API Reference</span>
					<ExternalLink class="h-3 w-3" />
				</a>
				{#if auth.isAuthenticated}
					<a href="/dashboard" class="text-xs font-medium text-accent transition-colors hover:text-accent/80">Dashboard</a>
				{:else if !auth.loading}
					<a href="/login" class="text-xs font-medium text-accent transition-colors hover:text-accent/80">Sign In</a>
				{/if}
			</div>
		</div>
	</header>

	<main class="mx-auto max-w-3xl px-4 lg:px-8">
		<!-- Hero -->
		<section class="border-b border-border/50 pb-8 pt-10">
			<p class="mb-2 font-mono text-xs font-medium uppercase tracking-wider text-accent">Webhooks</p>
			<h1 class="mb-2 text-2xl font-bold leading-tight tracking-tight text-foreground">
				Webhook signing & verification
			</h1>
			<p class="max-w-xl text-sm leading-relaxed text-muted-foreground">
				Outbound notifications for incidents and agent state changes. Optionally signed with HMAC-SHA256 for integrity verification and replay protection.
			</p>
		</section>

		<!-- TOC -->
		<section class="border-b border-border/50 py-6">
			<h2 class="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">In this guide</h2>
			<div class="grid gap-1.5 sm:grid-cols-2">
				{#each [
					['#payload', '01', 'Payload format'],
					['#signing', '02', 'Signature headers'],
					['#verify', '03', 'Verification recipes'],
					['#replay', '04', 'Replay protection'],
					['#unsigned', '05', 'Unsigned webhooks'],
					['#secret', '06', 'Generating a secret']
				] as [href, num, label]}
					<a {href} class="flex items-center gap-2 py-1 text-sm text-foreground transition-colors hover:text-accent">
						<span class="font-mono text-xs text-accent">{num}</span> {label}
					</a>
				{/each}
			</div>
		</section>

		<!-- 01 Payload -->
		<section id="payload" class="scroll-mt-16 border-b border-border/50 py-10">
			<div class="mb-4 flex items-center gap-3">
				<span class="flex h-8 w-8 items-center justify-center rounded-md bg-accent/10 font-mono text-sm font-bold text-accent">1</span>
				<h2 class="text-lg font-semibold text-foreground">Payload format</h2>
			</div>
			<p class="mb-4 text-sm leading-relaxed text-muted-foreground">
				Two payload shapes — incidents and agent events. Content-Type is always
				<code class="border border-border bg-muted/40 px-1 font-mono text-xs">application/json</code>.
			</p>

			<h3 class="mb-2 text-sm font-medium text-foreground">Incident events</h3>
			<pre class="mb-3 overflow-x-auto border border-border bg-muted/30 p-3 font-mono text-xs leading-relaxed text-foreground"><code>{`{
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
}`}</code></pre>
			<p class="mb-6 text-xs text-muted-foreground">
				Event types: <code class="font-mono">incident.opened</code>, <code class="font-mono">incident.resolved</code>. For resolved events, <code class="font-mono">resolved_at</code> is non-null.
			</p>

			<h3 class="mb-2 text-sm font-medium text-foreground">Agent events</h3>
			<pre class="mb-3 overflow-x-auto border border-border bg-muted/30 p-3 font-mono text-xs leading-relaxed text-foreground"><code>{`{
  "event_type": "agent.offline",
  "timestamp": "2026-04-15T12:34:56Z",
  "agent_id": "770e8400-e29b-41d4-a716-446655440002",
  "agent_name": "agent-prod-1",
  "affected_monitors": 3
}`}</code></pre>
			<p class="text-xs text-muted-foreground">
				Event types: <code class="font-mono">agent.offline</code>, <code class="font-mono">agent.online</code>, <code class="font-mono">agent.maintenance</code>. Fields vary by event type.
			</p>
		</section>

		<!-- 02 Signing -->
		<section id="signing" class="scroll-mt-16 border-b border-border/50 py-10">
			<div class="mb-4 flex items-center gap-3">
				<span class="flex h-8 w-8 items-center justify-center rounded-md bg-accent/10 font-mono text-sm font-bold text-accent">2</span>
				<h2 class="text-lg font-semibold text-foreground">Signature headers</h2>
			</div>
			<p class="mb-4 text-sm leading-relaxed text-muted-foreground">
				When a signing secret is configured on the webhook channel, every request includes three headers:
			</p>
			<div class="mb-4 overflow-x-auto">
				<table class="w-full text-xs">
					<thead>
						<tr class="border-b border-border text-left text-muted-foreground">
							<th class="py-2 pr-4 font-medium">Header</th>
							<th class="py-2 font-medium">Value</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border/40">
						<tr>
							<td class="py-2 pr-4 font-mono text-foreground">X-Watchdog-Signature-256</td>
							<td class="py-2 font-mono text-muted-foreground"><code>sha256=&lt;hex&gt;</code> — HMAC-SHA256 of the signed string</td>
						</tr>
						<tr>
							<td class="py-2 pr-4 font-mono text-foreground">X-Watchdog-Timestamp</td>
							<td class="py-2 text-muted-foreground">Unix seconds at send time</td>
						</tr>
						<tr>
							<td class="py-2 pr-4 font-mono text-foreground">X-Watchdog-Nonce</td>
							<td class="py-2 text-muted-foreground">UUIDv4, fresh per request</td>
						</tr>
					</tbody>
				</table>
			</div>

			<h3 class="mb-2 text-sm font-medium text-foreground">Signed string construction</h3>
			<pre class="mb-3 overflow-x-auto border border-border bg-muted/30 p-3 font-mono text-xs text-foreground"><code>{'{timestamp}.{nonce}.{body}'}</code></pre>
			<p class="mb-4 text-xs text-muted-foreground">
				Concatenated with literal <code class="font-mono">.</code> separators. <code class="font-mono">body</code> is the raw JSON payload bytes.
			</p>

			<h3 class="mb-2 text-sm font-medium text-foreground">Algorithm</h3>
			<ul class="ml-4 list-disc space-y-1 text-xs text-muted-foreground">
				<li>HMAC-SHA256</li>
				<li>Key: the configured signing secret (treated as UTF-8 bytes)</li>
				<li>Output: lowercase hex, 64 characters</li>
			</ul>
		</section>

		<!-- 03 Verify -->
		<section id="verify" class="scroll-mt-16 border-b border-border/50 py-10">
			<div class="mb-4 flex items-center gap-3">
				<span class="flex h-8 w-8 items-center justify-center rounded-md bg-accent/10 font-mono text-sm font-bold text-accent">3</span>
				<h2 class="text-lg font-semibold text-foreground">Verification recipes</h2>
			</div>

			<h3 class="mb-2 text-sm font-medium text-foreground">Go</h3>
			<pre class="mb-6 overflow-x-auto border border-border bg-muted/30 p-3 font-mono text-xs leading-relaxed text-foreground"><code>{`package main

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
}`}</code></pre>

			<h3 class="mb-2 text-sm font-medium text-foreground">Python</h3>
			<pre class="mb-6 overflow-x-auto border border-border bg-muted/30 p-3 font-mono text-xs leading-relaxed text-foreground"><code>{`import hmac, hashlib, time
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
    return "", 200`}</code></pre>

			<h3 class="mb-2 text-sm font-medium text-foreground">Node.js</h3>
			<pre class="overflow-x-auto border border-border bg-muted/30 p-3 font-mono text-xs leading-relaxed text-foreground"><code>{`import crypto from "node:crypto";
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
        .update(\`\${ts}.\${nonce}.\`)
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
});`}</code></pre>
		</section>

		<!-- 04 Replay -->
		<section id="replay" class="scroll-mt-16 border-b border-border/50 py-10">
			<div class="mb-4 flex items-center gap-3">
				<span class="flex h-8 w-8 items-center justify-center rounded-md bg-accent/10 font-mono text-sm font-bold text-accent">4</span>
				<h2 class="text-lg font-semibold text-foreground">Replay protection</h2>
			</div>
			<p class="mb-4 text-sm leading-relaxed text-muted-foreground">
				Replay protection is enforced by the receiver, not the sender. The sender provides enough context to enable it:
			</p>
			<ul class="ml-4 list-disc space-y-2 text-sm leading-relaxed text-muted-foreground">
				<li>
					<strong class="text-foreground">Timestamp</strong> — reject requests outside a freshness window (5 minutes recommended). Longer windows increase replay exposure; shorter windows risk false rejection from clock drift.
				</li>
				<li>
					<strong class="text-foreground">Nonce</strong> — dedup per request. Use a TTL-bounded cache (Redis, in-memory with expiry) sized to the freshness window. After the window, a replayed nonce is already rejected by the timestamp check.
				</li>
			</ul>
			<p class="mt-4 text-sm leading-relaxed text-muted-foreground">
				Running both checks (timestamp <em>and</em> nonce) is defense-in-depth. The timestamp check alone is insufficient because an attacker within the window can replay. The nonce check alone requires permanent storage.
			</p>
		</section>

		<!-- 05 Unsigned -->
		<section id="unsigned" class="scroll-mt-16 border-b border-border/50 py-10">
			<div class="mb-4 flex items-center gap-3">
				<span class="flex h-8 w-8 items-center justify-center rounded-md bg-accent/10 font-mono text-sm font-bold text-accent">5</span>
				<h2 class="text-lg font-semibold text-foreground">Unsigned webhooks</h2>
			</div>
			<p class="text-sm leading-relaxed text-muted-foreground">
				When no signing secret is configured, the signature headers are absent. The payload format is otherwise identical. Use HTTPS and origin-IP allowlisting for minimum security; prefer signing for untrusted network paths.
			</p>
		</section>

		<!-- 06 Generating a secret -->
		<section id="secret" class="scroll-mt-16 py-10">
			<div class="mb-4 flex items-center gap-3">
				<span class="flex h-8 w-8 items-center justify-center rounded-md bg-accent/10 font-mono text-sm font-bold text-accent">6</span>
				<h2 class="text-lg font-semibold text-foreground">Generating a secret</h2>
			</div>
			<p class="mb-4 text-sm leading-relaxed text-muted-foreground">
				From the UI: Settings → Alert Channels → Create/Edit Webhook → Generate.
			</p>
			<p class="mb-3 text-sm leading-relaxed text-muted-foreground">
				Generates a 256-bit (32-byte) random value, hex-encoded (64 characters). Equivalent shell command:
			</p>
			<pre class="mb-4 overflow-x-auto border border-border bg-muted/30 p-3 font-mono text-xs text-foreground"><code>openssl rand -hex 32</code></pre>
			<p class="text-sm leading-relaxed text-muted-foreground">
				Store the secret securely on the receiver side. The sender stores it encrypted (AES-GCM).
			</p>
		</section>

		<!-- Footer -->
		<footer class="border-t border-border/50 py-6 text-center">
			<a href="/docs" class="inline-flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground">
				<span>Back to API Reference</span>
				<ExternalLink class="h-3 w-3" />
			</a>
		</footer>
	</main>
</div>
