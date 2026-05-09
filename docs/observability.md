# Observability

Language: **English** | [简体中文](observability.zh-CN.md)

SnowPanel currently exposes backend metrics, structured request logs, health
checks, readiness checks, audit logs, and core-agent tracing logs.

## HTTP Health

- `GET /health` reports whether the backend process is alive.
- `GET /ready` reports dependency readiness, including database and core-agent connectivity.
- `GET /metrics` exposes Prometheus metrics from the backend.

## Backend Metrics

The backend exports Prometheus metrics with the `snowpanel` namespace:

- `snowpanel_http_requests_total`
  - labels: `method`, `route`, `status`
  - counts HTTP requests handled by Gin
- `snowpanel_http_request_duration_seconds`
  - labels: `method`, `route`
  - records HTTP request latency
- `snowpanel_http_requests_in_flight`
  - tracks in-flight HTTP requests
- `snowpanel_agent_requests_total`
  - labels: `outcome`, `transport`
  - counts backend to core-agent gRPC calls
- `snowpanel_agent_request_duration_seconds`
  - labels: `outcome`, `transport`
  - records core-agent gRPC call latency

Example scrape target:

```yaml
scrape_configs:
  - job_name: snowpanel-backend
    static_configs:
      - targets:
          - backend:8080
```

## Logs And Correlation

- Every backend request gets a request id through the request-id middleware.
- Access logs include `request_id`, method, path, status, latency, client IP, and user agent.
- Core-agent uses `tracing` logs and should be collected from the host-agent service logs in production.
- Audit logs record user-facing administrative actions and are available through the audit API and UI.

## Operational Checks

For production incidents, start with:

1. Check backend `/ready` to separate process liveness from dependency readiness.
2. Check `snowpanel_agent_requests_total{outcome="error"}` and the `transport` label for backend to agent failures.
3. Correlate backend access logs by `request_id`.
4. Inspect core-agent service logs on the host when agent transport errors increase.
5. Use audit logs to confirm who initiated state-changing operations.

## Current Scope

Prometheus metrics are implemented for backend HTTP and backend to core-agent
calls. Full distributed tracing is not required for the current milestone; if it
is added later, it should build on the existing request id and structured logs.
