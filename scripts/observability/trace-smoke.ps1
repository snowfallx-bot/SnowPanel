param(
  [Parameter(Mandatory = $true)]
  [string]$AccessToken,
  [string]$BackendBaseUrl = "http://127.0.0.1:8080",
  [string]$JaegerBaseUrl = "http://127.0.0.1:16686",
  [string]$RequestId = "",
  [int]$TraceWaitSeconds = 30
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

function New-RequestId {
  return "trace-e2e-$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
}

function Get-TraceServiceSet {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Trace
  )

  $serviceSet = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
  foreach ($span in $Trace.spans) {
    $process = $Trace.processes.$($span.processID)
    if ($null -ne $process -and -not [string]::IsNullOrWhiteSpace([string]$process.serviceName)) {
      [void]$serviceSet.Add([string]$process.serviceName)
    }
  }
  return $serviceSet
}

if ([string]::IsNullOrWhiteSpace($RequestId)) {
  $RequestId = New-RequestId
}

$requestStartUnixMicros = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() * 1000

$headers = @{
  Authorization = "Bearer $AccessToken"
  "X-Request-ID" = $RequestId
}

Write-Host "Triggering core-agent path via GET $BackendBaseUrl/api/v1/dashboard/summary ..."
$dashboardEnvelope = Invoke-ObservabilityJsonRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/dashboard/summary" -Headers $headers -ExpectedStatusCodes @(200)
if ($dashboardEnvelope.code -ne 0) {
  throw "Dashboard request returned non-zero code: $($dashboardEnvelope | ConvertTo-Json -Depth 10 -Compress)"
}

Write-Host "Dashboard request succeeded. request_id=$RequestId"
Write-Host "Polling Jaeger API for correlated trace (timeout=${TraceWaitSeconds}s) ..."

$deadline = (Get-Date).AddSeconds($TraceWaitSeconds)
$foundTrace = $null

while ((Get-Date) -lt $deadline) {
  $query = "$JaegerBaseUrl/api/traces?service=snowpanel-backend&lookback=1h&limit=50"
  $tracesEnvelope = Invoke-ObservabilityJsonRequest -Method "GET" -Uri $query
  foreach ($trace in $tracesEnvelope.data) {
    $serviceSet = Get-TraceServiceSet -Trace $trace
    $hasBackend = $serviceSet.Contains("snowpanel-backend")
    $hasCoreAgent = $serviceSet.Contains("snowpanel-core-agent")
    if (-not ($hasBackend -and $hasCoreAgent)) {
      continue
    }

    $latestSpanStart = 0
    foreach ($span in $trace.spans) {
      if ($span.startTime -gt $latestSpanStart) {
        $latestSpanStart = [int64]$span.startTime
      }
    }

    if ($latestSpanStart -lt $requestStartUnixMicros) {
      continue
    }

    $foundTrace = $trace
    break
  }

  if ($null -ne $foundTrace) {
    break
  }

  Start-Sleep -Seconds 2
}

if ($null -eq $foundTrace) {
  throw "No cross-service trace found in Jaeger within ${TraceWaitSeconds}s. Check OTEL env vars, collector health, and Jaeger ingestion."
}

$serviceNames = (Get-TraceServiceSet -Trace $foundTrace).ToArray() | Sort-Object
Write-Host "Trace validation passed."
Write-Host "trace_id: $($foundTrace.traceID)"
Write-Host "services: $($serviceNames -join ', ')"
