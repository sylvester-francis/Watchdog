# OpenTelemetry Traces

The hub can emit distributed traces in OTLP format to any OTel-compatible receiver. Disabled by default; opt in with one environment variable.

## Enable

```sh
WATCHDOG_OTEL_ENABLED=true
WATCHDOG_OTEL_SERVICE_NAME=watchdog-hub        # optional; default "watchdog-hub"
OTEL_EXPORTER_OTLP_ENDPOINT=https://otel.example.com   # required when enabled
```

When `WATCHDOG_OTEL_ENABLED` is unset or false, the hub installs a no-op tracer — no exporter is created, no network egress, zero cost.

## Configuration

The hub owns only two env vars:

| Variable | Purpose |
|---|---|
| `WATCHDOG_OTEL_ENABLED` | Gate. `true` initializes the SDK; anything else uses the no-op tracer. |
| `WATCHDOG_OTEL_SERVICE_NAME` | Sets the `service.name` resource attribute. Default `watchdog-hub`. |

Everything else — endpoint, headers, sampler, timeouts, TLS — flows through the standard `OTEL_*` environment variables read directly by the OTel SDK. See the [OTel spec](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) for the full list. The most common ones:

| Variable | Purpose |
|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Where to send spans. The OTLP HTTP exporter assumes `/v1/traces` is appended automatically. |
| `OTEL_EXPORTER_OTLP_HEADERS` | Auth headers, e.g. `Authorization=Bearer <token>` or `api-key=...`. Comma-separated `k=v` list. |
| `OTEL_EXPORTER_OTLP_TRACES_INSECURE` | Set to `true` for plaintext HTTP (local collector only). |
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

- Every HTTP request is wrapped in a server span (`otelecho` middleware).
- Spans propagate W3C trace context, so traces stitch across services that share the propagator.

## What's not instrumented yet (planned)

- Outbound DB queries (pgx instrumentation)
- WebSocket heartbeat processing
- Workflow job execution
- Agent-side spans (separate milestone)
- Structured-log export (separate milestone)
- OTLP-format metrics alongside the existing `/metrics` Prometheus endpoint

## Verifying the pipeline

```sh
WATCHDOG_OTEL_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
OTEL_EXPORTER_OTLP_TRACES_INSECURE=true \
./hub
```

Send a few requests, then check the receiver UI for spans tagged `service.name=watchdog-hub`. If nothing arrives:

- Confirm the receiver is listening on the endpoint port (default OTLP HTTP is 4318).
- Check hub logs for `OTLP HTTP trace exporter` errors.
- Try `curl -v <endpoint>/v1/traces` from the hub host to confirm reachability.
- Verify firewall / egress rules allow outbound HTTP/S to the receiver.
