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
$SupportsSkipHttpErrorCheck = (Get-Command Invoke-WebRequest).Parameters.ContainsKey("SkipHttpErrorCheck")

function New-RequestId {
  return "trace-e2e-$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
}

function Invoke-JsonRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Method,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [hashtable]$Headers = @{},
    [object]$Body = $null,
    [int[]]$ExpectedStatusCodes = @(200)
  )

  $requestParams = @{
    Method             = $Method
    Uri                = $Uri
    Headers            = $Headers
  }

  if ($SupportsSkipHttpErrorCheck) {
    $requestParams.SkipHttpErrorCheck = $true
  }

  if ($null -ne $Body) {
    $requestParams.ContentType = "application/json"
    $requestParams.Body = ($Body | ConvertTo-Json -Depth 10 -Compress)
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
$dashboardEnvelope = Invoke-JsonRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/dashboard/summary" -Headers $headers -ExpectedStatusCodes @(200)
if ($dashboardEnvelope.code -ne 0) {
  throw "Dashboard request returned non-zero code: $($dashboardEnvelope | ConvertTo-Json -Depth 10 -Compress)"
}

Write-Host "Dashboard request succeeded. request_id=$RequestId"
Write-Host "Polling Jaeger API for correlated trace (timeout=${TraceWaitSeconds}s) ..."

$deadline = (Get-Date).AddSeconds($TraceWaitSeconds)
$foundTrace = $null

while ((Get-Date) -lt $deadline) {
  $query = "$JaegerBaseUrl/api/traces?service=snowpanel-backend&lookback=1h&limit=50"
  $tracesEnvelope = Invoke-JsonRequest -Method "GET" -Uri $query
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
