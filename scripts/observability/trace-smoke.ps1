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

function Get-ResponseHeaderValue {
  param(
    [Parameter(Mandatory = $true)]
    [hashtable]$Headers,
    [Parameter(Mandatory = $true)]
    [string]$Name
  )

  foreach ($headerName in $Headers.Keys) {
    if ([string]$headerName -ieq $Name) {
      return [string]$Headers[$headerName]
    }
  }

  return ""
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

function Get-TraceSpanServiceName {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Trace,
    [Parameter(Mandatory = $true)]
    [object]$Span
  )

  $process = $Trace.processes.$($Span.processID)
  if ($null -eq $process) {
    return ""
  }

  return [string]$process.serviceName
}

function Get-TraceTagValue {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Span,
    [Parameter(Mandatory = $true)]
    [string]$TagKey
  )

  if ($null -eq $Span.tags) {
    return ""
  }

  foreach ($tag in $Span.tags) {
    if ([string]$tag.key -ieq $TagKey) {
      return [string]$tag.value
    }
  }

  return ""
}

function Trace-HasRequestIdForService {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Trace,
    [Parameter(Mandatory = $true)]
    [string]$ServiceName,
    [Parameter(Mandatory = $true)]
    [string]$RequestId
  )

  foreach ($span in $Trace.spans) {
    if ((Get-TraceSpanServiceName -Trace $Trace -Span $span) -ine $ServiceName) {
      continue
    }

    if ((Get-TraceTagValue -Span $span -TagKey "snowpanel.request_id") -eq $RequestId) {
      return $true
    }
  }

  return $false
}

function Get-CoreAgentGrpcMethodsForRequest {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Trace,
    [Parameter(Mandatory = $true)]
    [string]$RequestId
  )

  $methods = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
  foreach ($span in $Trace.spans) {
    if ((Get-TraceSpanServiceName -Trace $Trace -Span $span) -ine "snowpanel-core-agent") {
      continue
    }

    if ((Get-TraceTagValue -Span $span -TagKey "snowpanel.request_id") -ne $RequestId) {
      continue
    }

    $grpcMethod = Get-TraceTagValue -Span $span -TagKey "grpc.method"
    if (-not [string]::IsNullOrWhiteSpace($grpcMethod)) {
      [void]$methods.Add($grpcMethod)
    }
  }

  return $methods
}

if ([string]::IsNullOrWhiteSpace($RequestId)) {
  $RequestId = New-RequestId
}

$requestStartUnixMicros = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() * 1000
$requiredCoreGrpcMethods = @(
  "/snowpanel.agent.v1.SystemService/GetSystemOverview",
  "/snowpanel.agent.v1.SystemService/GetRealtimeResource"
)

$headers = @{
  Authorization = "Bearer $AccessToken"
  "X-Request-ID" = $RequestId
}

Write-Host "Triggering core-agent path via GET $BackendBaseUrl/api/v1/dashboard/summary ..."
$dashboardResponse = Invoke-ObservabilityApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/dashboard/summary" -Headers $headers -ExpectedStatusCodes @(200)
if ($null -eq $dashboardResponse.Json -or [int]$dashboardResponse.Json.code -ne 0) {
  throw "Dashboard request returned non-zero code: $($dashboardResponse.Content)"
}

$responseRequestId = Get-ResponseHeaderValue -Headers $dashboardResponse.Headers -Name "X-Request-ID"
if ([string]::IsNullOrWhiteSpace($responseRequestId)) {
  throw "Dashboard response did not include X-Request-ID header."
}
if ($responseRequestId -ne $RequestId) {
  throw "Dashboard response X-Request-ID mismatch. expected='$RequestId' actual='$responseRequestId'"
}

Write-Host "Dashboard request succeeded. request_id=$RequestId"
Write-Host "Polling Jaeger API for correlated trace (timeout=${TraceWaitSeconds}s) ..."

$foundTrace = $null
$foundCoreGrpcMethods = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
$lastObservation = "No traces fetched yet."

try {
  Wait-ObservabilityCondition -Description "Jaeger correlated trace" -TimeoutSeconds $TraceWaitSeconds -TimeoutMessage "No request-correlated cross-service trace found in Jaeger within ${TraceWaitSeconds}s. Check OTEL env vars, collector health, and Jaeger ingestion." -Check {
    $query = "$JaegerBaseUrl/api/traces?service=snowpanel-backend&lookback=1h&limit=100"
    $tracesEnvelope = Invoke-ObservabilityJsonRequest -Method "GET" -Uri $query
    if ($null -eq $tracesEnvelope -or $null -eq $tracesEnvelope.data) {
      $lastObservation = "Jaeger traces API returned empty body."
      return $false
    }

    if ($tracesEnvelope.data.Count -eq 0) {
      $lastObservation = "Jaeger returned 0 traces."
      return $false
    }

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

      if (-not (Trace-HasRequestIdForService -Trace $trace -ServiceName "snowpanel-backend" -RequestId $RequestId)) {
        $lastObservation = "Found cross-service trace $($trace.traceID) after request start, but backend span missing snowpanel.request_id=$RequestId."
        continue
      }

      $coreGrpcMethods = Get-CoreAgentGrpcMethodsForRequest -Trace $trace -RequestId $RequestId
      $missingMethods = @()
      foreach ($requiredMethod in $requiredCoreGrpcMethods) {
        if (-not $coreGrpcMethods.Contains($requiredMethod)) {
          $missingMethods += $requiredMethod
        }
      }
      if ($missingMethods.Count -gt 0) {
        $observedMethods = ($coreGrpcMethods.ToArray() | Sort-Object) -join ", "
        if ([string]::IsNullOrWhiteSpace($observedMethods)) {
          $observedMethods = "(none)"
        }
        $lastObservation = "Found trace $($trace.traceID) with request_id=$RequestId, but missing expected core-agent grpc.method tags: $($missingMethods -join ', '). Observed: $observedMethods"
        continue
      }

      $foundTrace = $trace
      $foundCoreGrpcMethods = $coreGrpcMethods
      return $true
    }

    $lastObservation = "Jaeger returned $($tracesEnvelope.data.Count) traces, but none matched request_id=$RequestId with required cross-service spans."
    return $false
  }
} catch {
  throw "$($_.Exception.Message) Last observation: $lastObservation"
}

$serviceNames = (Get-TraceServiceSet -Trace $foundTrace).ToArray() | Sort-Object
$coreMethods = ($foundCoreGrpcMethods.ToArray() | Sort-Object) -join ", "
Write-Host "Trace validation passed."
Write-Host "trace_id: $($foundTrace.traceID)"
Write-Host "services: $($serviceNames -join ', ')"
Write-Host "core grpc methods: $coreMethods"
Write-Host "request_id: $RequestId"
