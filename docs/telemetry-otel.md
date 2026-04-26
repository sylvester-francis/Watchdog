# OpenTelemetry Traces and Logs

The hub emits distributed traces and structured logs in OTLP format to any OTel-compatible receiver. Telemetry is enabled by default but only activates when an OTLP endpoint is configured — without one, a no-op provider is installed (no exporter, no network egress, no log spam).

## Enable

Set an OTLP endpoint and you're done:

```sh
OTEL_EXPORTER_OTLP_ENDPOINT=https://otel.example.com
```

That's it. Traces and logs both export there. The hub auto-detects the endpoint via the standard OTel env var.

To force-disable the SDK even when an endpoint is configured (e.g. CI / local dev sharing prod env files):

```sh
WATCHDOG_OTEL_ENABLED=false
```

## Configuration

The hub owns only two env vars:

| Variable | Default | Purpose |
|---|---|---|
| `WATCHDOG_OTEL_ENABLED` | `true` | Force-disable switch. Set to `false` to suppress the SDK even when an endpoint is configured. |
| `WATCHDOG_OTEL_SERVICE_NAME` | `watchdog-hub` | Sets the `service.name` resource attribute on every emitted trace and log record. |

The SDK is constructed iff `WATCHDOG_OTEL_ENABLED=true` AND a relevant OTLP endpoint env var is set. Each signal (traces, logs) checks independently — you can route them to different endpoints.

Everything else — endpoint, headers, sampler, timeouts, TLS — flows through the standard `OTEL_*` environment variables read directly by the OTel SDK. See the [OTel spec](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) for the full list. The most common ones:

| Variable | Purpose |
|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Where to send both signals. The OTLP HTTP exporter appends `/v1/traces` and `/v1/logs` automatically. |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Per-signal override for traces only. |
| `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` | Per-signal override for logs only. |
| `OTEL_EXPORTER_OTLP_HEADERS` | Auth headers, e.g. `Authorization=Bearer <token>` or `api-key=...`. Comma-separated `k=v` list. |
| `OTEL_EXPORTER_OTLP_TRACES_INSECURE` | Set to `true` for plaintext HTTP traces (local collector only). |
| `OTEL_EXPORTER_OTLP_LOGS_INSECURE` | Set to `true` for plaintext HTTP logs (local collector only). |
| `OTEL_TRACES_SAMPLER` | `always_on` (default), `always_off`, `traceidratio`, `parentbased_traceidratio`. |
| `OTEL_TRACES_SAMPLER_ARG` | Ratio for `traceidratio`, e.g. `0.1` for 10% sampling. |
| `OTEL_RESOURCE_ATTRIBUTES` | Extra resource attributes, e.g. `deployment.environment=prod,host.name=hub-01`. |

## Receiver recipes

### Local Tempo (Grafana stack)

```sh
WATCHDOG_OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://tempo:4318
OTEL_EXPORTER_OTLP_TRACES_INSECURE=true
```

### Jaeger (with built-in OTLP receiver, v1.35+)

```sh
WATCHDOG_OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
OTEL_EXPORTER_OTLP_TRACES_INSECURE=true
```

### Grafana Cloud

```sh
WATCHDOG_OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp-gateway-prod-<region>.grafana.net/otlp
OTEL_EXPORTER_OTLP_HEADERS=Authorization=Basic <base64(instance_id:api_token)>
```

### Datadog (OTLP HTTP receiver enabled)

```sh
WATCHDOG_OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=https://trace.agent.datadoghq.com
OTEL_EXPORTER_OTLP_HEADERS=DD-API-KEY=<api_key>
```

Datadog's OTLP receiver requires the agent's OTLP feature to be turned on at the agent or platform level — see Datadog docs.

### Honeycomb

```sh
WATCHDOG_OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=https://api.honeycomb.io
OTEL_EXPORTER_OTLP_HEADERS=x-honeycomb-team=<api_key>
```

## What's instrumented today

- **Traces**: every HTTP request is wrapped in a server span (`otelecho` middleware). Spans propagate W3C trace context, so traces stitch across services that share the propagator.
- **Logs**: every `slog` log record (Info, Warn, Error, Debug) is emitted to both stdout/stderr (Docker-friendly) AND the OTel logs exporter when an endpoint is configured. The bridge captures structured attributes (`slog.String`, `slog.Int`, etc.) as OTel log attributes.

## What's not instrumented yet (planned)

- Outbound DB queries (pgx instrumentation)
- WebSocket heartbeat processing
- Workflow job execution
- Agent-side spans and logs (separate milestone)
- OTLP-format metrics alongside the existing `/metrics` Prometheus endpoint

## Verifying the pipeline

```sh
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
OTEL_EXPORTER_OTLP_TRACES_INSECURE=true \
OTEL_EXPORTER_OTLP_LOGS_INSECURE=true \
./hub
```

Send a few requests. Check the receiver UI for spans tagged `service.name=watchdog-hub` and log records with the same service name. If nothing arrives:

- Confirm the receiver is listening on the endpoint port (default OTLP HTTP is 4318).
- Check hub logs for `OTLP HTTP trace exporter` or `OTLP HTTP log exporter` errors.
- Try `curl -v <endpoint>/v1/traces` and `<endpoint>/v1/logs` from the hub host to confirm reachability.
- Verify firewall / egress rules allow outbound HTTP/S to the receiver.
- Confirm `WATCHDOG_OTEL_ENABLED` is not explicitly set to `false`.
