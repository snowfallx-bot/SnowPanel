param(
  [string]$PrometheusBaseUrl = "http://127.0.0.1:9090"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

$RequiredRecordingRules = @(
  "snowpanel:backend_http_total:rate5m",
  "snowpanel:backend_http_5xx:rate5m",
  "snowpanel:backend_http_availability:ratio5m",
  "snowpanel:core_agent_grpc_error_ratio:ratio5m"
)

$RequiredAlertRules = @(
  "SnowPanelBackendDown",
  "SnowPanelCoreAgentMetricsDown",
  "SnowPanelBackendP95LatencyHigh",
  "SnowPanelBackendP95LatencyCritical",
  "SnowPanelCoreAgentP95LatencyHigh",
  "SnowPanelCoreAgentP95LatencyCritical",
  "SnowPanelBackendAgentTransportErrorsHigh",
  "SnowPanelCoreAgentGrpcErrorRateHigh",
  "SnowPanelCoreAgentGrpcErrorRateCritical",
  "SnowPanelCoreAgentInFlightHigh",
  "SnowPanelBackendAvailabilitySLOWarning",
  "SnowPanelBackendAvailabilitySLOCritical"
)

Write-Host "Checking Prometheus loaded rules via $PrometheusBaseUrl/api/v1/rules ..."
$rulesEnvelope = Invoke-ObservabilityJsonRequest -Method "GET" -Uri "$PrometheusBaseUrl/api/v1/rules"

if ($rulesEnvelope.status -ne "success") {
  throw "Prometheus rules API returned non-success status: $($rulesEnvelope | ConvertTo-Json -Depth 10 -Compress)"
}

$recordingLoaded = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::Ordinal)
$alertLoaded = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::Ordinal)

foreach ($group in $rulesEnvelope.data.groups) {
  foreach ($rule in $group.rules) {
    if ($rule.type -eq "recording") {
      [void]$recordingLoaded.Add([string]$rule.name)
    } elseif ($rule.type -eq "alerting") {
      [void]$alertLoaded.Add([string]$rule.name)
    }
  }
}

$missingRecordings = @()
foreach ($ruleName in $RequiredRecordingRules) {
  if (-not $recordingLoaded.Contains($ruleName)) {
    $missingRecordings += $ruleName
  }
}

$missingAlerts = @()
foreach ($ruleName in $RequiredAlertRules) {
  if (-not $alertLoaded.Contains($ruleName)) {
    $missingAlerts += $ruleName
  }
}

if ($missingRecordings.Count -gt 0 -or $missingAlerts.Count -gt 0) {
  $parts = @()
  if ($missingRecordings.Count -gt 0) {
    $parts += "missing recording rules: $($missingRecordings -join ', ')"
  }
  if ($missingAlerts.Count -gt 0) {
    $parts += "missing alert rules: $($missingAlerts -join ', ')"
  }
  throw "Prometheus rules smoke failed: $($parts -join ' | ')"
}

Write-Host "Prometheus rules smoke passed."
Write-Host "recording rules checked: $($RequiredRecordingRules.Count)"
Write-Host "alert rules checked: $($RequiredAlertRules.Count)"
