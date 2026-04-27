# Observability Scripts

Utility scripts for validating SnowPanel observability plumbing in environments where Docker/host-agent stacks are already running.

Windows note: if local execution policy blocks scripts, run with:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File <script.ps1> ...
```

## `trace-smoke.ps1`

Trigger one backend request that proxies to core-agent, then verify Jaeger contains a recent trace with both services.

```powershell
pwsh -File ./scripts/observability/trace-smoke.ps1 `
  -AccessToken "<access_token>" `
  -BackendBaseUrl "http://127.0.0.1:8080" `
  -JaegerBaseUrl "http://127.0.0.1:16686"
```

Key parameters:

- `AccessToken` (required): bearer token for protected API calls
- `RequestId` (optional): override generated request id
- `TraceWaitSeconds` (optional): Jaeger polling timeout (default `30`)

## `validate-config.ps1`

Validate Prometheus and Alertmanager configs via `promtool`/`amtool` (Docker images by default, with local binary fallback when Docker is unavailable), including `promtool test rules` for alert rule unit tests.

```powershell
pwsh -File ./scripts/observability/validate-config.ps1
```

## `generate-alertmanager-config.ps1`

Generate a concrete Alertmanager production config with real webhook channels.

```powershell
pwsh -File ./scripts/observability/generate-alertmanager-config.ps1 `
  -WarningWebhookUrl "https://ops.example.com/alerts/warning" `
  -CriticalWebhookUrl "https://ops.example.com/alerts/critical" `
  -CriticalEscalationWebhookUrl "https://oncall.example.com/paging/critical" `
  -OutputPath "deploy/observability/alertmanager/alertmanager.generated.yml"
```

Key parameters:

- `WarningWebhookUrl` (required): warning notification webhook
- `CriticalWebhookUrl` (required): critical notification webhook
- `CriticalEscalationWebhookUrl` (optional): extra escalation webhook for critical alerts
- `OutputPath` (optional): generated config path (default `deploy/observability/alertmanager/alertmanager.generated.yml`)

## `prometheus-rules-smoke.ps1`

Validate that required recording and alert rules are loaded in a running Prometheus instance.

```powershell
pwsh -File ./scripts/observability/prometheus-rules-smoke.ps1 `
  -PrometheusBaseUrl "http://127.0.0.1:9090"
```

## `alertmanager-smoke.ps1`

Inject a synthetic alert into Alertmanager and verify it is observable via Alertmanager API.

```powershell
pwsh -File ./scripts/observability/alertmanager-smoke.ps1 `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093" `
  -AlertName "SnowPanelSmokeAlert" `
  -Severity "critical"
```

Key parameters:

- `AlertmanagerBaseUrl` (optional): Alertmanager endpoint (default `http://127.0.0.1:9093`)
- `AlertDurationSeconds` (optional): synthetic alert lifetime (default `120`)
- `WaitSeconds` (optional): wait timeout for API visibility (default `60`)

## `full-smoke.ps1`

Run both trace and alertmanager validations in one command.

```powershell
pwsh -File ./scripts/observability/full-smoke.ps1 `
  -AccessToken "<access_token>" `
  -BackendBaseUrl "http://127.0.0.1:8080" `
  -JaegerBaseUrl "http://127.0.0.1:16686" `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093"
```

Or let the script log in and fetch token automatically:

```powershell
pwsh -File ./scripts/observability/full-smoke.ps1 `
  -LoginUsername "admin" `
  -LoginPassword "<admin_password>" `
  -BackendBaseUrl "http://127.0.0.1:8080" `
  -JaegerBaseUrl "http://127.0.0.1:16686" `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093"
```
