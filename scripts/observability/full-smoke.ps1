param(
  [Parameter(Mandatory = $true)]
  [string]$AccessToken,
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

$traceScript = Join-Path $PSScriptRoot "trace-smoke.ps1"
$alertScript = Join-Path $PSScriptRoot "alertmanager-smoke.ps1"

$traceArgs = @{
  AccessToken      = $AccessToken
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
