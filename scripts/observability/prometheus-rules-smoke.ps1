param(
  [string]$PrometheusBaseUrl = "http://127.0.0.1:9090"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest
$SupportsSkipHttpErrorCheck = (Get-Command Invoke-WebRequest).Parameters.ContainsKey("SkipHttpErrorCheck")

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

function Invoke-JsonRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Method,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [int[]]$ExpectedStatusCodes = @(200)
  )

  $requestParams = @{
    Method = $Method
    Uri    = $Uri
  }

  if ($SupportsSkipHttpErrorCheck) {
    $requestParams.SkipHttpErrorCheck = $true
  }

  try {
    $response = Invoke-WebRequest @requestParams
  } catch {
    if ($SupportsSkipHttpErrorCheck) {
      throw
    }
    $exceptionResponse = $_.Exception.Response
    if ($null -eq $exceptionResponse) {
      throw
    }

    $stream = $exceptionResponse.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($stream)
    $content = $reader.ReadToEnd()
    $reader.Dispose()
    $stream.Dispose()

    $response = [PSCustomObject]@{
      StatusCode = [int]$exceptionResponse.StatusCode
      Content    = $content
    }
  }

  if ($response.StatusCode -notin $ExpectedStatusCodes) {
    throw "Expected status $($ExpectedStatusCodes -join ', ') from $Method $Uri, got $($response.StatusCode). Body: $($response.Content)"
  }

  if ([string]::IsNullOrWhiteSpace($response.Content)) {
    return $null
  }

  return $response.Content | ConvertFrom-Json -Depth 20
}

Write-Host "Checking Prometheus loaded rules via $PrometheusBaseUrl/api/v1/rules ..."
$rulesEnvelope = Invoke-JsonRequest -Method "GET" -Uri "$PrometheusBaseUrl/api/v1/rules"

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
