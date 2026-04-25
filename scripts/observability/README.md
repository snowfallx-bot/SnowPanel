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
- `WaitSeconds` (optional): wait timeout for API visibility (default `20`)

## `full-smoke.ps1`

Run both trace and alertmanager validations in one command.

```powershell
pwsh -File ./scripts/observability/full-smoke.ps1 `
  -AccessToken "<access_token>" `
  -BackendBaseUrl "http://127.0.0.1:8080" `
  -JaegerBaseUrl "http://127.0.0.1:16686" `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093"
```
