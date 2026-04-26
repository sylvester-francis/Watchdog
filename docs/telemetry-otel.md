# OpenTelemetry Traces, Logs, and Metrics

The hub emits distributed traces, structured logs, and runtime metrics in OTLP format to any OTel-compatible receiver. Traces and logs are gated on an OTLP endpoint being configured — without one, a no-op provider is installed (no exporter, no network egress, no log spam). Metrics always flow to the local Prometheus `/metrics` endpoint via the otelprom reader; the OTLP push exporter is added on top when an endpoint is configured.

## Enable

Set an OTLP endpoint and you're done:

```sh
OTEL_EXPORTER_OTLP_ENDPOINT=https://otel.example.com
```

That's it. Traces, logs, and metrics all export there. The hub auto-detects the endpoint via the standard OTel env var.

To force-disable the SDK even when an endpoint is configured (e.g. CI / local dev sharing prod env files):

```sh
WATCHDOG_OTEL_ENABLED=false
```

## Configuration

The hub owns only two env vars:

| Variable | Default | Purpose |
|---|---|---|
| `WATCHDOG_OTEL_ENABLED` | `true` | Force-disable switch. Set to `false` to suppress the OTLP push exporters for traces, logs, and metrics even when an endpoint is configured. The Prometheus `/metrics` reader is unaffected — it always runs. |
| `WATCHDOG_OTEL_SERVICE_NAME` | `watchdog-hub` | Sets the `service.name` resource attribute on every emitted trace, log record, and metric. |

The OTLP exporter for each signal is constructed iff `WATCHDOG_OTEL_ENABLED=true` AND a relevant OTLP endpoint env var is set. Each signal (traces, logs, metrics) checks independently — you can route them to different endpoints.

Everything else — endpoint, headers, sampler, timeouts, TLS — flows through the standard `OTEL_*` environment variables read directly by the OTel SDK. See the [OTel spec](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) for the full list. The most common ones:

| Variable | Purpose |
|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Where to send all three signals. The OTLP HTTP exporter appends `/v1/traces`, `/v1/logs`, and `/v1/metrics` automatically. |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Per-signal override for traces only. |
| `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` | Per-signal override for logs only. |
| `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` | Per-signal override for metrics only. |
| `OTEL_EXPORTER_OTLP_HEADERS` | Auth headers, e.g. `Authorization=Bearer <token>` or `api-key=...`. Comma-separated `k=v` list. |
| `OTEL_EXPORTER_OTLP_TRACES_INSECURE` | Set to `true` for plaintext HTTP traces (local collector only). |
| `OTEL_EXPORTER_OTLP_LOGS_INSECURE` | Set to `true` for plaintext HTTP logs (local collector only). |
| `OTEL_EXPORTER_OTLP_METRICS_INSECURE` | Set to `true` for plaintext HTTP metrics (local collector only). |
| `OTEL_METRIC_EXPORT_INTERVAL` | Push cadence for OTLP metrics in milliseconds. Default `60000` (60s). |
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
- **Metrics**: HTTP request latency (`watchdog_http_request_duration_seconds`), heartbeat processing latency (`watchdog_heartbeat_processing_seconds`), active WebSocket agents (`watchdog_ws_connections_active`), DB pool acquired connections (`watchdog_db_pool_active_connections`), and active incidents by status (`watchdog_incidents_active`). The OTel meter is the single source of truth — values flow out to the Prometheus `/metrics` endpoint via the otelprom reader and to OTLP receivers via the periodic push reader when telemetry is enabled.

## What's not instrumented yet (planned)

- Outbound DB queries (pgx instrumentation)
- WebSocket heartbeat processing spans
- Workflow job execution
- Agent-side spans, logs, and metrics (separate milestone)

## Verifying the pipeline

```sh
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
OTEL_EXPORTER_OTLP_TRACES_INSECURE=true \
OTEL_EXPORTER_OTLP_LOGS_INSECURE=true \
OTEL_EXPORTER_OTLP_METRICS_INSECURE=true \
./hub
```

Send a few requests. Check the receiver UI for spans, log records, and metric data points tagged `service.name=watchdog-hub`. The Prometheus `/metrics` endpoint surfaces the same metrics locally (admin session required) regardless of OTLP configuration. If nothing arrives:

- Confirm the receiver is listening on the endpoint port (default OTLP HTTP is 4318).
- Check hub logs for `OTLP HTTP trace exporter`, `OTLP HTTP log exporter`, or `OTLP HTTP metric exporter` errors.
- Try `curl -v <endpoint>/v1/traces`, `<endpoint>/v1/logs`, and `<endpoint>/v1/metrics` from the hub host to confirm reachability.
- Verify firewall / egress rules allow outbound HTTP/S to the receiver.
- Confirm `WATCHDOG_OTEL_ENABLED` is not explicitly set to `false`.
- For metrics, remember the default push interval is 60s — give the periodic reader a full cycle before assuming the pipeline is broken.
