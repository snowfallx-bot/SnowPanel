[CmdletBinding(DefaultParameterSetName = "token")]
param(
  [Parameter(Mandatory = $true, ParameterSetName = "token")]
  [string]$AccessToken,
  [Parameter(Mandatory = $true, ParameterSetName = "login")]
  [string]$LoginPassword,
  [Parameter(ParameterSetName = "login")]
  [string]$LoginUsername = "admin",
  [string]$BackendBaseUrl = "http://127.0.0.1:8080",
  [string]$JaegerBaseUrl = "http://127.0.0.1:16686",
  [string]$AlertmanagerBaseUrl = "http://127.0.0.1:9093",
  [string]$RequestId = "",
  [int]$TraceWaitSeconds = 30,
  [string]$AlertName = "SnowPanelSmokeAlert",
  [ValidateSet("critical", "warning")]
  [string]$Severity = "critical",
  [int]$AlertDurationSeconds = 120,
  [int]$AlertWaitSeconds = 20
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$commonScript = Join-Path $PSScriptRoot "common.ps1"
$traceScript = Join-Path $PSScriptRoot "trace-smoke.ps1"
$alertScript = Join-Path $PSScriptRoot "alertmanager-smoke.ps1"
. $commonScript

$resolvedAccessToken = $AccessToken
if ($PSCmdlet.ParameterSetName -eq "login") {
  Write-Host "Logging in via $BackendBaseUrl/api/v1/auth/login ..."
  $loginEnvelope = Invoke-ObservabilityJsonRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/login" -Body @{
    username = $LoginUsername
    password = $LoginPassword
  } -ExpectedStatusCodes @(200)

  if ($loginEnvelope.code -ne 0) {
    throw "Login returned non-zero code: $($loginEnvelope | ConvertTo-Json -Depth 10 -Compress)"
  }

  if ($loginEnvelope.data.user.must_change_password -eq $true) {
    throw "Login succeeded but user must change password before protected APIs are allowed. Rotate password first, then rerun full smoke."
  }

  $resolvedAccessToken = [string]$loginEnvelope.data.access_token
  if ([string]::IsNullOrWhiteSpace($resolvedAccessToken)) {
    throw "Login response did not include access_token."
  }
}

$traceArgs = @{
  AccessToken      = $resolvedAccessToken
  BackendBaseUrl   = $BackendBaseUrl
  JaegerBaseUrl    = $JaegerBaseUrl
  TraceWaitSeconds = $TraceWaitSeconds
}

if (-not [string]::IsNullOrWhiteSpace($RequestId)) {
  $traceArgs.RequestId = $RequestId
}

$alertArgs = @{
  AlertmanagerBaseUrl = $AlertmanagerBaseUrl
  AlertName           = $AlertName
  Severity            = $Severity
  AlertDurationSeconds = $AlertDurationSeconds
  WaitSeconds         = $AlertWaitSeconds
}

Write-Host "Running trace smoke validation ..."
& $traceScript @traceArgs

Write-Host "Running alertmanager smoke validation ..."
& $alertScript @alertArgs

Write-Host "Observability full smoke passed (trace + alertmanager)."
