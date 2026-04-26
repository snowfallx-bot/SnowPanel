param(
  [ValidateSet("container-agent", "host-agent")]
  [string]$AgentMode = "container-agent",
  [string]$HostAgentTarget = "host.docker.internal:50051",
  [string]$HostAgentMetricsBaseUrl = "http://127.0.0.1:9108",
  [string]$FailureLogDir = ".github/workflow-logs/observability-smoke"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$ProjectName = "snowpanel-obsv-smoke"
$BackendPort = "18083"
$PrometheusPort = "19090"
$AlertmanagerPort = "19093"
$JaegerPort = "16687"
$BackendBaseUrl = "http://127.0.0.1:$BackendPort"
$PrometheusBaseUrl = "http://127.0.0.1:$PrometheusPort"
$JaegerBaseUrl = "http://127.0.0.1:$JaegerPort"
$AlertmanagerBaseUrl = "http://127.0.0.1:$AlertmanagerPort"
$HostAgentMetricsEndpoint = $HostAgentMetricsBaseUrl.TrimEnd("/")
if ($HostAgentMetricsEndpoint -notmatch "/metrics$") {
  $HostAgentMetricsEndpoint = "$HostAgentMetricsEndpoint/metrics"
}
$ResolvedFailureLogDir = $FailureLogDir
if (-not [string]::IsNullOrWhiteSpace($ResolvedFailureLogDir) -and -not [System.IO.Path]::IsPathRooted($ResolvedFailureLogDir)) {
  $ResolvedFailureLogDir = Join-Path $RepoRoot $ResolvedFailureLogDir
}
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
if ($AgentMode -eq "container-agent") {
  $ComposeArgs += @("--profile", "container-agent")
}
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
    if ([string]::IsNullOrWhiteSpace($HostAgentTarget)) {
      throw "HostAgentTarget must not be empty when AgentMode=host-agent."
    }
    $env:AGENT_TARGET = $HostAgentTarget

    Wait-UntilReady -Description "host core-agent metrics endpoint" -Attempts 30 -DelaySeconds 1 -Check {
      $response = Invoke-ApiRequest -Method "GET" -Uri $HostAgentMetricsEndpoint -ExpectedStatusCodes @(200)
      return $response.StatusCode -eq 200
    }
  } else {
    Remove-Item -Path Env:AGENT_TARGET -ErrorAction SilentlyContinue
  }

  $services = @("postgres", "redis", "backend", "jaeger", "otel-collector", "alertmanager", "prometheus")
  if ($AgentMode -eq "container-agent") {
    $services = @("postgres", "redis", "core-agent", "backend", "jaeger", "otel-collector", "alertmanager", "prometheus")
  }

  Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments (@("up", "-d", "--build") + $services)

  Wait-BackendReadyJson -BackendBaseUrl $BackendBaseUrl

  Wait-UntilReady -Description "jaeger ui" -Check {
    $response = Invoke-ApiRequest -Method "GET" -Uri $JaegerBaseUrl -ExpectedStatusCodes @(200)
    return $response.StatusCode -eq 200
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
    -AlertWaitSeconds 30 `
    -ValidateAllAlertSeverities `
    -ValidateInhibition

  $Completed = $true
  Write-Host "Observability smoke test passed."
} finally {
  if (-not $Completed) {
    Write-Host "Observability smoke test failed, printing compose status and logs..."
    Show-ComposeDiagnostics -ComposeArgs $ComposeArgs

    if (-not [string]::IsNullOrWhiteSpace($ResolvedFailureLogDir)) {
      New-Item -ItemType Directory -Force -Path $ResolvedFailureLogDir | Out-Null
      $composePsLogFile = Join-Path $ResolvedFailureLogDir "compose-ps.$AgentMode.log"
      $composeLogsFile = Join-Path $ResolvedFailureLogDir "compose-logs.$AgentMode.log"

      try {
        $composePsArgs = $ComposeArgs + @("ps")
        (& docker @composePsArgs 2>&1) | Out-File -FilePath $composePsLogFile -Encoding utf8
      } catch {
        Write-Warning "Failed to capture compose ps log: $($_.Exception.Message)"
      }

      try {
        $composeLogsArgs = $ComposeArgs + @("logs", "--no-color", "--timestamps", "--tail", "400")
        (& docker @composeLogsArgs 2>&1) | Out-File -FilePath $composeLogsFile -Encoding utf8
      } catch {
        Write-Warning "Failed to capture compose logs: $($_.Exception.Message)"
      }
    }
  }

  Stop-ComposeStack -ComposeArgs $ComposeArgs
}
