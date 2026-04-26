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
$ComposeArgs = @(
  "compose",
  "--project-name", $ProjectName,
  "-f", "docker-compose.yml",
  "-f", "docker-compose.observability.yml"
)
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

  Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("up", "-d", "--build", "postgres", "redis", "core-agent", "backend", "jaeger", "otel-collector", "alertmanager", "prometheus")

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

  $loginEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/login" -Body @{
    username = "admin"
    password = $BootstrapPassword
  }
  Assert-True ($loginEnvelope.code -eq 0) "Bootstrap login failed"
  Assert-True ($loginEnvelope.data.user.must_change_password -eq $true) "Bootstrap admin should require password rotation"

  $bootstrapToken = [string]$loginEnvelope.data.access_token
  $bootstrapHeaders = @{ Authorization = "Bearer $bootstrapToken" }

  $changePasswordEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/change-password" -Headers $bootstrapHeaders -Body @{
    current_password = $BootstrapPassword
    new_password = $RotatedPassword
  }
  Assert-True ($changePasswordEnvelope.code -eq 0) "Password rotation failed"
  Assert-True ($changePasswordEnvelope.data.user.must_change_password -eq $false) "Rotated user should clear must_change_password"

  $activeToken = [string]$changePasswordEnvelope.data.access_token
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
    -Instance "smoke-warning-route" `
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
