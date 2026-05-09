# Alerting Runbook

Language: **English** | [简体中文](alerting-runbook.zh-CN.md)

This runbook defines the operational ownership and first-response path for SnowPanel alerts.

## Ownership

| Severity | Owner | Policy |
| --- | --- | --- |
| `warning` | SnowPanel platform owner | Non-paging. Triage during working hours or the next maintenance window. |
| `critical` | SnowPanel on-call owner | Paging. Acknowledge immediately and escalate when service impact is confirmed or the issue is not mitigated within 15 minutes. |
| `critical` escalation | SnowPanel incident lead | Paging escalation. Used when the primary critical channel has not resolved or acknowledged the incident. |

Receiver mapping:

- `snowpanel-warning`: warning channel
- `snowpanel-critical`: primary critical paging channel
- `snowpanel-critical-escalation`: secondary critical escalation channel

## Routing And Dedup

- Grouping: `alertname`, `severity`, `instance`
- Warning cadence: `group_wait=30s`, `group_interval=10m`, `repeat_interval=4h`
- Critical cadence: `group_wait=10s`, `group_interval=2m`, `repeat_interval=30m`
- Critical escalation cadence: `group_wait=2m`, `group_interval=10m`, `repeat_interval=15m`
- Inhibition: a `critical` alert suppresses the matching `warning` alert when `alertname` and `instance` are equal.

Production configs should be generated with:

```powershell
pwsh -File ./scripts/observability/generate-alertmanager-config.ps1 `
  -WarningWebhookUrl "https://alerts.example.com/snowpanel-warning" `
  -CriticalWebhookUrl "https://alerts.example.com/snowpanel-critical" `
  -CriticalEscalationWebhookUrl "https://alerts.example.com/snowpanel-critical-escalation" `
  -OutputPath "deploy/observability/alertmanager/alertmanager.generated.yml"
```

Validate before rollout:

```powershell
pwsh -File ./scripts/observability/validate-config.ps1
```

## Synthetic Alert Test

Run warning and critical route checks against a running Alertmanager:

```powershell
pwsh -File ./scripts/observability/alertmanager-smoke.ps1 `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093" `
  -Severity warning

pwsh -File ./scripts/observability/alertmanager-smoke.ps1 `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093" `
  -Severity critical
```

Run inhibition validation:

```powershell
pwsh -File ./scripts/observability/alertmanager-inhibition-smoke.ps1 `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093"
```

## Triage

### `SnowPanelBackendDown`

1. Check backend container or service status.
2. Check `GET /health` and `GET /ready`.
3. Check backend logs for startup errors, database connection errors, and panic recovery entries.
4. Confirm Prometheus can reach backend `/metrics`.
5. If backend is crash-looping after a deployment, roll back the backend image or config change.

### `SnowPanelCoreAgentMetricsDown`

1. Identify whether the deployment is compose-agent or host-agent mode.
2. Compose mode: check `snowpanel-core-agent` container status and logs.
3. Host-agent mode: check the host `core-agent` systemd service and `CORE_AGENT_METRICS_*` bind settings.
4. Confirm firewall rules allow Prometheus to reach the metrics endpoint.
5. If metrics are down but `/ready` reports `agent=up`, treat this as observability degradation; if `/ready` reports `agent=down`, treat it as operational impact.

### Backend p95 latency high

1. Check whether latency is isolated to one route or all backend traffic.
2. Compare backend CPU/memory and database health.
3. Check agent RPC latency metrics to identify whether the backend is waiting on core-agent.
4. Use recent traces in Jaeger to find the slow span.
5. If the issue follows a release, roll back the backend or agent change.

### Core-agent gRPC error ratio high

1. Check backend agent RPC metrics by `rpc`, `outcome`, and `transport`.
2. Check core-agent logs with the same `request_id` from backend logs.
3. Confirm safe-root, service whitelist, Docker, and cron allowlist errors are expected denials rather than infrastructure failures.
4. If transport errors are elevated, verify host reachability, firewall, and core-agent process health.
5. Roll back recent agent config or binary changes when errors correlate with deployment.

## Correlation

Use `X-Request-ID` to connect user-visible failures to backend and core-agent logs:

1. Capture `X-Request-ID` from the HTTP response header or frontend error details.
2. Search backend logs for the same request ID.
3. Search core-agent logs for the same request ID.
4. In Jaeger, search recent traces for `snowpanel.request_id=<request_id>`.
5. Confirm the trace contains both `snowpanel-backend` and `snowpanel-core-agent` spans for agent-backed operations.

## Temporary Silence

Only silence alerts when an owner is actively working the issue or during a planned maintenance window.

Recommended labels:

- `alertname`
- `instance`
- `severity`

Keep silence duration short:

- Warning maintenance: up to 4 hours
- Critical mitigation: 30 to 60 minutes, then reassess

Record the reason, owner, and rollback or follow-up ticket in the silence comment.

## Rollback

1. Revert the generated Alertmanager config to the last known-good file.
2. Run `pwsh -File ./scripts/observability/validate-config.ps1`.
3. Restart or reload Alertmanager through the deployment mechanism.
4. Run warning and critical synthetic alert smoke tests.
5. Confirm resolved notifications are delivered when applicable.

Local development may continue to use the no-op baseline receiver. Production must use real warning and critical receivers before claiming alert delivery readiness.
