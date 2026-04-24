# Observability Notes

Language: **English** | [简体中文](observability.zh-CN.md)

## Scope

This document describes SnowPanel's current production troubleshooting baseline for:

- Metrics (Prometheus format)
- Cross-service request correlation (`X-Request-ID`)
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

Agent RPC metrics are labeled by:

- `rpc`
- `outcome` (`success` / `error`)
- `transport` (`true` / `false`)

Core-agent gRPC metrics are labeled by:

- `grpc_method`
- `outcome` (`ok` / `error`)

## Prometheus Baseline Stack

Repository now includes a baseline Prometheus deployment:

- Compose override: `docker-compose.observability.yml`
- Prometheus scrape config: `deploy/observability/prometheus/prometheus.yml`
- Alert rules: `deploy/observability/prometheus/alerts/snowpanel-alerts.yml`

Start baseline stack:

- Compose mode: `make up-observability`
- Host-agent mode: `make up-host-agent-observability`

Inspect Prometheus:

- `http://127.0.0.1:${PROMETHEUS_PORT:-9090}`

Stop:

- Compose mode: `make down-observability`
- Host-agent mode: `make down-host-agent-observability`

Notes:

- Baseline scrape targets assume backend `:8080` and core-agent metrics `:9108`.
- If your runtime ports differ, update `deploy/observability/prometheus/prometheus.yml` accordingly.

## Request Correlation

SnowPanel now propagates request IDs through the backend to core-agent:

1. Backend HTTP middleware accepts incoming `X-Request-ID` (or generates one).
2. The request ID is attached to request context and returned in response headers.
3. Backend gRPC client forwards it as gRPC metadata `x-request-id`.
4. Core-agent logs each gRPC call with:
   - `request_id`
   - `grpc_method`

This allows a single request path to be traced from browser/API client logs to backend logs and into core-agent logs.

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
- `SnowPanelCoreAgentP95LatencyHigh`
- `SnowPanelBackendAgentTransportErrorsHigh`
- `SnowPanelCoreAgentGrpcErrorRateHigh`
- `SnowPanelCoreAgentInFlightHigh`

## Current Gaps

- No full OpenTelemetry pipeline yet.
- No distributed trace backend (Jaeger/Tempo/etc.) yet.
- No unified OTel collector/exporter strategy across backend + core-agent yet.
