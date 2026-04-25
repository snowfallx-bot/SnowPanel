# Observability Notes

Language: **English** | [简体中文](observability.zh-CN.md)

## Scope

This document describes SnowPanel's current production troubleshooting baseline for:

- Metrics (Prometheus format)
- Cross-service request correlation (`X-Request-ID`)
- Distributed tracing (OTLP -> OTel Collector -> Jaeger)
- Operational log lookup

## Metrics

Backend exposes Prometheus metrics at:

- `GET /metrics`

Core-agent also exposes a standalone Prometheus endpoint (when enabled):

- `GET http://<CORE_AGENT_METRICS_HOST>:<CORE_AGENT_METRICS_PORT>/metrics`

Current key metric families include:

- `snowpanel_http_requests_total`
- `snowpanel_http_request_duration_seconds`
- `snowpanel_http_requests_in_flight`
- `snowpanel_agent_requests_total`
- `snowpanel_agent_request_duration_seconds`
- `snowpanel_core_agent_grpc_requests_total`
- `snowpanel_core_agent_grpc_request_duration_seconds`
- `snowpanel_core_agent_grpc_requests_in_flight`

Prometheus recording rules also derive SLO-oriented series:

- `snowpanel:backend_http_total:rate5m`
- `snowpanel:backend_http_5xx:rate5m`
- `snowpanel:backend_http_availability:ratio5m`
- `snowpanel:core_agent_grpc_error_ratio:ratio5m`

Agent RPC metrics are labeled by:

- `rpc`
- `outcome` (`success` / `error`)
- `transport` (`true` / `false`)

Core-agent gRPC metrics are labeled by:

- `grpc_method`
- `outcome` (`ok` / `error`)

## Observability Baseline Stack

Repository now includes a baseline observability deployment:

- Compose override: `docker-compose.observability.yml`
- Prometheus scrape config: `deploy/observability/prometheus/prometheus.yml`
- Alert rules: `deploy/observability/prometheus/alerts/snowpanel-alerts.yml`
- Alertmanager routing config: `deploy/observability/alertmanager/alertmanager.yml`
- OTel Collector config: `deploy/observability/otel-collector/config.yaml`
- Jaeger UI: `http://127.0.0.1:${JAEGER_UI_PORT:-16686}`

Start baseline stack:

- Compose mode: `make up-observability`
- Host-agent mode: `make up-host-agent-observability`

Inspect UIs:

- `http://127.0.0.1:${PROMETHEUS_PORT:-9090}`
- `http://127.0.0.1:${ALERTMANAGER_PORT:-9093}`
- `http://127.0.0.1:${JAEGER_UI_PORT:-16686}`

Stop:

- Compose mode: `make down-observability`
- Host-agent mode: `make down-host-agent-observability`

Notes:

- Baseline scrape targets assume backend `:8080` and core-agent metrics `:9108`.
- If your runtime ports differ, update `deploy/observability/prometheus/prometheus.yml` accordingly.
- Baseline Alertmanager receivers are intentionally no-op. Configure real warning/critical webhook/email/slack receivers in `deploy/observability/alertmanager/alertmanager.yml`.
- Compose observability mode enables OTLP tracing export for `backend` and containerized `core-agent` by default.
- In host-agent mode, also set OTEL variables in `deploy/core-agent/systemd/core-agent.env.example` (or `/etc/snowpanel/core-agent.env`) so host `core-agent` exports traces to the collector.
- Before smoke/runtime checks, run `pwsh -File ./scripts/observability/validate-config.ps1` to fail fast on Prometheus/Alertmanager config errors.

## Request Correlation

SnowPanel now propagates request IDs through the backend to core-agent:

1. Backend HTTP middleware accepts incoming `X-Request-ID` (or generates one).
2. The request ID is attached to request context and returned in response headers.
3. Backend gRPC client forwards it as gRPC metadata `x-request-id`.
4. Core-agent logs each gRPC call with:
   - `request_id`
   - `grpc_method`

This allows a single request path to be traced from browser/API client logs to backend logs and into core-agent logs.

## Distributed Tracing Baseline

Current trace path:

1. Backend HTTP requests create server spans.
2. Backend gRPC client creates child spans and propagates W3C trace context to core-agent.
3. Core-agent extracts remote context and creates gRPC server spans under the same trace.
4. Both services export OTLP traces to OTel Collector.
5. Collector batches and forwards traces to Jaeger.

Recommended OTEL variables:

- `OTEL_TRACING_ENABLED=true`
- `OTEL_EXPORTER_OTLP_ENDPOINT=<collector-host>:4317`
- `OTEL_EXPORTER_OTLP_INSECURE=true`
- `OTEL_TRACES_SAMPLER_ARG=1.0`

Default service names:

- backend: `snowpanel-backend`
- core-agent: `snowpanel-core-agent`

## Tracing Validation Checklist

Use this checklist to verify the trace chain in either compose mode or host-agent mode.

1. Start stack:
   - Compose mode: `make up-observability`
   - Host-agent mode: `make up-host-agent-observability`
2. Log in and keep a valid bearer token (for protected APIs under `/api/v1/*`).
3. Call one backend route that must proxy to core-agent, for example:
   - `GET /api/v1/dashboard/summary`
   - Include a custom header such as `X-Request-ID: trace-e2e-001`
4. Confirm backend response still contains the same `X-Request-ID`.
5. In Jaeger (`http://127.0.0.1:${JAEGER_UI_PORT:-16686}`), verify a single trace contains spans from both services:
   - `snowpanel-backend`
   - `snowpanel-core-agent`
6. If using host-agent mode, also confirm host `core-agent` OTEL vars are set correctly in `/etc/snowpanel/core-agent.env` (or from `deploy/core-agent/systemd/core-agent.env.example`).

Optional helper script (PowerShell):

```powershell
pwsh -File ./scripts/observability/trace-smoke.ps1 `
  -AccessToken "<access_token>" `
  -BackendBaseUrl "http://127.0.0.1:8080" `
  -JaegerBaseUrl "http://127.0.0.1:16686"
```

The script triggers `GET /api/v1/dashboard/summary` with a generated `X-Request-ID`, then polls Jaeger and fails unless it finds a recent trace containing both `snowpanel-backend` and `snowpanel-core-agent`.

See also: [`scripts/observability/README.md`](../scripts/observability/README.md) for both tracing and Alertmanager smoke script usage.
For a one-shot check, you can run `pwsh -File ./scripts/observability/full-smoke.ps1 -AccessToken "<access_token>"`.
`full-smoke.ps1` also supports automatic login mode via `-LoginUsername` + `-LoginPassword` if you do not want to fetch token manually.

## Fast Triage Flow

1. Capture the `X-Request-ID` from browser devtools or API response headers.
2. Search backend logs by `request_id`.
3. Search core-agent logs by the same `request_id`.
4. Check `/metrics` for elevated:
   - `snowpanel_http_request_duration_seconds`
   - `snowpanel_agent_request_duration_seconds`
   - `snowpanel_agent_requests_total{outcome="error",...}`
5. Check core-agent metrics for method-level pressure:
   - `snowpanel_core_agent_grpc_requests_total`
   - `snowpanel_core_agent_grpc_request_duration_seconds`
   - `snowpanel_core_agent_grpc_requests_in_flight`

## Baseline Alerts

Default alerts include:

- `SnowPanelBackendDown`
- `SnowPanelCoreAgentMetricsDown`
- `SnowPanelBackendP95LatencyHigh`
- `SnowPanelBackendP95LatencyCritical`
- `SnowPanelCoreAgentP95LatencyHigh`
- `SnowPanelCoreAgentP95LatencyCritical`
- `SnowPanelBackendAgentTransportErrorsHigh`
- `SnowPanelCoreAgentGrpcErrorRateHigh`
- `SnowPanelCoreAgentGrpcErrorRateCritical`
- `SnowPanelCoreAgentInFlightHigh`
- `SnowPanelBackendAvailabilitySLOWarning`
- `SnowPanelBackendAvailabilitySLOCritical`

## Alert Delivery Baseline

Prometheus forwards alerts to Alertmanager (`alertmanager:9093`) by default.

Current default routing:

- `severity="warning"` -> `snowpanel-warning`
- `severity="critical"` -> `snowpanel-critical`

Both `snowpanel-warning` and `snowpanel-critical` ship as template no-op receivers with commented webhook examples so teams can wire real notification channels explicitly.

## Alertmanager Rollout Checklist

Use this checklist when moving from baseline no-op routing to real production delivery.

1. Replace both `snowpanel-warning` and `snowpanel-critical` receivers with real channels in `deploy/observability/alertmanager/alertmanager.yml` (webhook/email/slack/wechat/etc).
   - You can start from `deploy/observability/alertmanager/alertmanager.production.example.yml` and then apply your real channel endpoints/secrets.
2. Keep explicit route ownership by severity:
   - `critical` -> paging channel
   - `warning` -> non-paging ops channel (add dedicated route/receiver if needed)
3. Keep (or extend) inhibition rules so `critical` suppresses duplicate `warning` noise for the same `alertname`.
4. Roll out and validate routing:
   - `make up-observability` (or `make up-host-agent-observability`)
   - verify receiver/routing state in Alertmanager UI (`/#/status`)
5. Run a controlled delivery test:
   - either inject a synthetic alert:
     - `pwsh -File ./scripts/observability/alertmanager-smoke.ps1`
   - or temporarily lower one alert threshold in `deploy/observability/prometheus/alerts/snowpanel-alerts.yml`
   - then generate matching traffic/load as needed
   - confirm notification arrives once per dedup window and includes labels (`alertname`, `severity`, `instance`)
6. Restore the original threshold after validation and commit the final config/rule set.

## Current Gaps

- No browser/frontend tracing yet.
- No trace-backed log shipping pipeline; request correlation still mainly relies on logs + `X-Request-ID`.
- Alert routing, deduplication, escalation, and SLO/SLA calibration still need production tuning.
