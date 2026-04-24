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

Current key metric families include:

- `snowpanel_http_requests_total`
- `snowpanel_http_request_duration_seconds`
- `snowpanel_http_requests_in_flight`
- `snowpanel_agent_requests_total`
- `snowpanel_agent_request_duration_seconds`

Agent RPC metrics are labeled by:

- `rpc`
- `outcome` (`success` / `error`)
- `transport` (`true` / `false`)

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

## Current Gaps

- No full OpenTelemetry pipeline yet.
- No distributed trace backend (Jaeger/Tempo/etc.) yet.
- No standalone Prometheus endpoint in core-agent yet (agent telemetry is currently surfaced via backend-side RPC metrics + agent logs).

