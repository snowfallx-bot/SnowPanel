param(
  [Parameter(Mandatory = $true)]
  [string]$AccessToken,
  [string]$BackendBaseUrl = "http://127.0.0.1:8080",
  [string]$JaegerBaseUrl = "http://127.0.0.1:16686",
  [string]$RequestId = "",
  [int]$TraceWaitSeconds = 30,
  [ValidateRange(1, 120)]
  [int]$TriggerRetryIntervalSeconds = 6
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

function Invoke-TraceTriggerRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$AccessToken,
    [Parameter(Mandatory = $true)]
    [string]$BackendBaseUrl,
    [Parameter(Mandatory = $true)]
    [string]$RequestId
  )

  $headers = @{
    Authorization = "Bearer $AccessToken"
    "X-Request-ID" = $RequestId
  }

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
}

function Get-JaegerTraceCandidates {
  param(
    [Parameter(Mandatory = $true)]
    [string]$JaegerBaseUrl,
    [int]$Limit = 150
  )

  $serviceQuery = "$JaegerBaseUrl/api/traces?service=snowpanel-backend&lookback=1h&limit=$Limit"
  $serviceEnvelope = Invoke-ObservabilityJsonRequest -Method "GET" -Uri $serviceQuery
  $serviceTraces = @()
  if ($null -ne $serviceEnvelope -and $null -ne $serviceEnvelope.data) {
    $serviceTraces = @($serviceEnvelope.data)
  }

  if ($serviceTraces.Count -gt 0) {
    return [PSCustomObject]@{
      Traces         = $serviceTraces
      Source         = "service"
      ServiceCount   = $serviceTraces.Count
      FallbackError  = ""
    }
  }

  $globalTraces = @()
  $fallbackError = ""
  try {
    $globalQuery = "$JaegerBaseUrl/api/traces?lookback=1h&limit=$Limit"
    $globalEnvelope = Invoke-ObservabilityJsonRequest -Method "GET" -Uri $globalQuery
    if ($null -ne $globalEnvelope -and $null -ne $globalEnvelope.data) {
      $globalTraces = @($globalEnvelope.data)
    }
  } catch {
    $fallbackError = $_.Exception.Message
  }

  return [PSCustomObject]@{
    Traces         = $globalTraces
    Source         = "service+global"
    ServiceCount   = 0
    FallbackError  = $fallbackError
  }
}

if ([string]::IsNullOrWhiteSpace($RequestId)) {
  $RequestId = New-RequestId
}

$requestStartUnixMicros = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() * 1000
$requiredCoreGrpcMethods = @(
  "/snowpanel.agent.v1.SystemService/GetSystemOverview",
  "/snowpanel.agent.v1.SystemService/GetRealtimeResource"
)
$triggerCount = 0
$nextTriggerAtSeconds = [double][Math]::Max(1, $TriggerRetryIntervalSeconds)
$triggerStopwatch = [System.Diagnostics.Stopwatch]::StartNew()

Write-Host "Triggering core-agent path via GET $BackendBaseUrl/api/v1/dashboard/summary ..."
Invoke-TraceTriggerRequest -AccessToken $AccessToken -BackendBaseUrl $BackendBaseUrl -RequestId $RequestId
$triggerCount++
Write-Host "Dashboard request succeeded. request_id=$RequestId trigger_count=$triggerCount"
Write-Host "Polling Jaeger API for correlated trace (timeout=${TraceWaitSeconds}s, retry_interval=${TriggerRetryIntervalSeconds}s) ..."

$foundTrace = $null
$foundCoreGrpcMethods = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
$lastObservation = "No traces fetched yet. trigger_count=0."

try {
  Wait-ObservabilityCondition -Description "Jaeger correlated trace" -TimeoutSeconds $TraceWaitSeconds -TimeoutMessage "No request-correlated cross-service trace found in Jaeger within ${TraceWaitSeconds}s. Check OTEL env vars, collector health, and Jaeger ingestion." -Check {
    if ($triggerStopwatch.Elapsed.TotalSeconds -ge $nextTriggerAtSeconds) {
      Invoke-TraceTriggerRequest -AccessToken $AccessToken -BackendBaseUrl $BackendBaseUrl -RequestId $RequestId
      $triggerCount++
      $nextTriggerAtSeconds += [double][Math]::Max(1, $TriggerRetryIntervalSeconds)
      Write-Host "Re-triggered dashboard request for trace capture. request_id=$RequestId trigger_count=$triggerCount"
    }

    $traceCandidates = Get-JaegerTraceCandidates -JaegerBaseUrl $JaegerBaseUrl
    $traces = @($traceCandidates.Traces)
    if ($traces.Count -eq 0) {
      $fallbackSuffix = ""
      if (-not [string]::IsNullOrWhiteSpace($traceCandidates.FallbackError)) {
        $fallbackSuffix = " global_fallback_error=$($traceCandidates.FallbackError)"
      }
      $lastObservation = "Jaeger returned 0 traces via $($traceCandidates.Source). trigger_count=$triggerCount.$fallbackSuffix"
      return $false
    }

    $validTraceCount = 0
    foreach ($trace in $traces) {
      if ($null -eq $trace -or $null -eq $trace.spans -or $null -eq $trace.processes) {
        continue
      }
      $validTraceCount++

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

    if ($validTraceCount -eq 0) {
      $lastObservation = "Jaeger returned $($traces.Count) trace envelopes, but none had valid spans/processes. trigger_count=$triggerCount."
      return $false
    }

    $lastObservation = "Jaeger returned $validTraceCount valid traces, but none matched request_id=$RequestId with required cross-service spans. trigger_count=$triggerCount."
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
Write-Host "trigger_count: $triggerCount"
