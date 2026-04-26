param(
  [ValidateSet("container-agent", "host-agent")]
  [string]$AgentMode = "container-agent",
  [string]$HostAgentTarget = "host.docker.internal:50051",
  [string]$HostAgentMetricsBaseUrl = "http://127.0.0.1:9108"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

$ProjectName = "snowpanel-obsv-smoke"
$BackendPort = "18083"
$PrometheusPort = "19090"
$AlertmanagerPort = "19093"
$JaegerPort = "16687"
$BackendBaseUrl = "http://127.0.0.1:$BackendPort"
$PrometheusBaseUrl = "http://127.0.0.1:$PrometheusPort"
$JaegerBaseUrl = "http://127.0.0.1:$JaegerPort"
$AlertmanagerBaseUrl = "http://127.0.0.1:$AlertmanagerPort"
$BootstrapPassword = "ObsSmokeBootstrap1!"
$RotatedPassword = "ObsSmokeRotated2!"
$JwtSecret = "ObservabilitySmokeSecret_2026_Check_123!"
$composeFiles = @(
  "docker-compose.yml",
  "docker-compose.observability.yml"
)
if ($AgentMode -eq "host-agent") {
  $composeFiles += "docker-compose.host-agent.yml"
}

$ComposeArgs = @(
  "compose",
  "--project-name", $ProjectName
)
$composeFiles | ForEach-Object {
  $ComposeArgs += @("-f", $_)
}
$Completed = $false

Assert-DockerAvailable -ScriptPath "scripts/ci/observability-smoke.ps1"

try {
  $validateScript = Join-Path $PSScriptRoot "..\observability\validate-config.ps1"
  & $validateScript

  $env:APP_ENV = "production"
  $env:BACKEND_PORT = $BackendPort
  $env:PROMETHEUS_PORT = $PrometheusPort
  $env:ALERTMANAGER_PORT = $AlertmanagerPort
  $env:JAEGER_UI_PORT = $JaegerPort
  $env:JWT_SECRET = $JwtSecret
  $env:DEFAULT_ADMIN_PASSWORD = $BootstrapPassword
  $env:LOGIN_ATTEMPT_STORE = "redis"
  if ($AgentMode -eq "host-agent") {
    $env:AGENT_TARGET = $HostAgentTarget

    Wait-UntilReady -Description "host core-agent metrics endpoint" -Attempts 30 -DelaySeconds 1 -Check {
      $response = Invoke-WebRequest -Method "GET" -Uri "$HostAgentMetricsBaseUrl/metrics" -SkipHttpErrorCheck
      if ([int]$response.StatusCode -ne 200) {
        return $false
      }
      return $response.Content -match "snowpanel_core_agent_grpc_requests_total"
    }
  }

  $services = @("postgres", "redis", "backend", "jaeger", "otel-collector", "alertmanager", "prometheus")
  if ($AgentMode -eq "container-agent") {
    $services = @("postgres", "redis", "core-agent", "backend", "jaeger", "otel-collector", "alertmanager", "prometheus")
  }

  Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments (@("up", "-d", "--build") + $services)

  Wait-BackendReadyJson -BackendBaseUrl $BackendBaseUrl

  Wait-UntilReady -Description "jaeger ui" -Check {
    $response = Invoke-WebRequest -Method "GET" -Uri $JaegerBaseUrl -SkipHttpErrorCheck
    return [int]$response.StatusCode -eq 200
  }

  Wait-UntilReady -Description "alertmanager api" -Check {
    $status = Invoke-JsonRequest -Method "GET" -Uri "$AlertmanagerBaseUrl/api/v2/status"
    return $null -ne $status
  }

  Wait-UntilReady -Description "prometheus rules api" -Check {
    $rules = Invoke-JsonRequest -Method "GET" -Uri "$PrometheusBaseUrl/api/v1/rules"
    return $null -ne $rules -and $rules.status -eq "success"
  }

  $rulesSmokeScript = Join-Path $PSScriptRoot "..\observability\prometheus-rules-smoke.ps1"
  & $rulesSmokeScript -PrometheusBaseUrl $PrometheusBaseUrl

  $bootstrapSession = Initialize-BootstrapAdminSession -ApiBaseUrl $BackendBaseUrl -BootstrapPassword $BootstrapPassword -RotatedPassword $RotatedPassword
  $activeToken = $bootstrapSession.RotatedAccessToken
  Assert-True (-not [string]::IsNullOrWhiteSpace($activeToken)) "Rotated session access token should not be empty"

  $fullSmokeScript = Join-Path $PSScriptRoot "..\observability\full-smoke.ps1"
  & $fullSmokeScript `
    -AccessToken $activeToken `
    -BackendBaseUrl $BackendBaseUrl `
    -JaegerBaseUrl $JaegerBaseUrl `
    -AlertmanagerBaseUrl $AlertmanagerBaseUrl `
    -TraceWaitSeconds 45 `
    -AlertWaitSeconds 30

  $alertSmokeScript = Join-Path $PSScriptRoot "..\observability\alertmanager-smoke.ps1"
  & $alertSmokeScript `
    -AlertmanagerBaseUrl $AlertmanagerBaseUrl `
    -AlertName "SnowPanelSmokeAlertWarning" `
    -Severity "warning" `
    -AlertDurationSeconds 120 `
    -WaitSeconds 30

  $Completed = $true
  Write-Host "Observability smoke test passed."
} finally {
  if (-not $Completed) {
    Write-Host "Observability smoke test failed, printing compose status and logs..."
    Show-ComposeDiagnostics -ComposeArgs $ComposeArgs
  }

  Stop-ComposeStack -ComposeArgs $ComposeArgs
}
