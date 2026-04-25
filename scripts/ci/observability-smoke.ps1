$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$ProjectName = "snowpanel-obsv-smoke"
$BackendPort = "18083"
$PrometheusPort = "19090"
$AlertmanagerPort = "19093"
$JaegerPort = "16687"
$BackendBaseUrl = "http://127.0.0.1:$BackendPort"
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

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  throw "docker command not found. Install Docker/Compose before running scripts/ci/observability-smoke.ps1."
}

function Invoke-Compose {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$Arguments
  )

  & docker @ComposeArgs @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "docker compose $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
  }
}

function Invoke-JsonRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Method,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [object]$Body = $null,
    [hashtable]$Headers = @{},
    [int[]]$ExpectedStatusCodes = @(200)
  )

  $requestParams = @{
    Method             = $Method
    Uri                = $Uri
    Headers            = $Headers
    SkipHttpErrorCheck = $true
  }

  if ($null -ne $Body) {
    $requestParams.ContentType = "application/json"
    $requestParams.Body = ($Body | ConvertTo-Json -Depth 10 -Compress)
  }

  $response = Invoke-WebRequest @requestParams
  if ($response.StatusCode -notin $ExpectedStatusCodes) {
    throw "Expected status $($ExpectedStatusCodes -join ', ') from $Method $Uri, got $($response.StatusCode). Body: $($response.Content)"
  }

  if ([string]::IsNullOrWhiteSpace($response.Content)) {
    return $null
  }

  return $response.Content | ConvertFrom-Json -Depth 20
}

function Assert-True {
  param(
    [Parameter(Mandatory = $true)]
    [bool]$Condition,
    [Parameter(Mandatory = $true)]
    [string]$Message
  )

  if (-not $Condition) {
    throw $Message
  }
}

function Wait-UntilReady {
  param(
    [Parameter(Mandatory = $true)]
    [scriptblock]$Check,
    [Parameter(Mandatory = $true)]
    [string]$Description,
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
    try {
      if (& $Check) {
        return
      }
    } catch {
      if ($attempt -eq $Attempts) {
        throw "Timed out waiting for $Description. Last error: $($_.Exception.Message)"
      }
    }
    Start-Sleep -Seconds $DelaySeconds
  }

  throw "Timed out waiting for $Description"
}

try {
  $env:APP_ENV = "production"
  $env:BACKEND_PORT = $BackendPort
  $env:PROMETHEUS_PORT = $PrometheusPort
  $env:ALERTMANAGER_PORT = $AlertmanagerPort
  $env:JAEGER_UI_PORT = $JaegerPort
  $env:JWT_SECRET = $JwtSecret
  $env:DEFAULT_ADMIN_PASSWORD = $BootstrapPassword
  $env:LOGIN_ATTEMPT_STORE = "redis"

  Invoke-Compose -Arguments @("up", "-d", "--build", "postgres", "redis", "core-agent", "backend", "jaeger", "otel-collector", "alertmanager", "prometheus")

  Wait-UntilReady -Description "backend readiness" -Check {
    $ready = Invoke-JsonRequest -Method "GET" -Uri "$BackendBaseUrl/ready"
    return $ready.code -eq 0 -and $ready.data.checks.database -eq "up" -and $ready.data.checks.agent -eq "up"
  }

  Wait-UntilReady -Description "jaeger ui" -Check {
    $response = Invoke-WebRequest -Method "GET" -Uri $JaegerBaseUrl -SkipHttpErrorCheck
    return [int]$response.StatusCode -eq 200
  }

  Wait-UntilReady -Description "alertmanager api" -Check {
    $status = Invoke-JsonRequest -Method "GET" -Uri "$AlertmanagerBaseUrl/api/v2/status"
    return $null -ne $status
  }

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

  $Completed = $true
  Write-Host "Observability smoke test passed."
} finally {
  if (-not $Completed) {
    Write-Host "Observability smoke test failed, printing compose status and logs..."
    try {
      Invoke-Compose -Arguments @("ps")
    } catch {
      Write-Warning $_.Exception.Message
    }
    try {
      Invoke-Compose -Arguments @("logs", "--no-color", "--tail", "200")
    } catch {
      Write-Warning $_.Exception.Message
    }
  }

  try {
    Invoke-Compose -Arguments @("down", "-v", "--remove-orphans")
  } catch {
    Write-Warning $_.Exception.Message
  }
}
