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
  foreach ($span in (Get-TraceSpans -Trace $Trace)) {
    if ($null -eq $span.processID) {
      continue
    }
    $process = Get-TraceProcessById -Trace $Trace -ProcessId ([string]$span.processID)
    if ($null -ne $process -and -not [string]::IsNullOrWhiteSpace([string]$process.serviceName)) {
      [void]$serviceSet.Add([string]$process.serviceName)
    }
  }
  return ,$serviceSet
}

function Get-TraceSpanServiceName {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Trace,
    [Parameter(Mandatory = $true)]
    [object]$Span
  )

  if ($null -eq $Span.processID) {
    return ""
  }

  $process = Get-TraceProcessById -Trace $Trace -ProcessId ([string]$Span.processID)
  if ($null -eq $process) {
    return ""
  }

  return [string]$process.serviceName
}

function Get-TraceSpans {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Trace
  )

  if ($null -eq $Trace -or $null -eq $Trace.spans) {
    return @()
  }

  if ($Trace.spans -is [string]) {
    return @()
  }

  $spans = @()
  if ($Trace.spans -is [System.Collections.IEnumerable]) {
    foreach ($span in $Trace.spans) {
      if ($null -ne $span) {
        $spans += $span
      }
    }
    return $spans
  }

  return @($Trace.spans)
}

function Get-TraceProcessById {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Trace,
    [Parameter(Mandatory = $true)]
    [string]$ProcessId
  )

  if ($null -eq $Trace -or $null -eq $Trace.processes -or [string]::IsNullOrWhiteSpace($ProcessId)) {
    return $null
  }

  if ($Trace.processes -is [System.Collections.IDictionary]) {
    return $Trace.processes[$ProcessId]
  }

  $namedProperty = $Trace.processes.PSObject.Properties[$ProcessId]
  if ($null -ne $namedProperty) {
    return $namedProperty.Value
  }

  foreach ($property in $Trace.processes.PSObject.Properties) {
    if ([string]$property.Name -eq $ProcessId) {
      return $property.Value
    }
  }

  return $null
}

function Get-UniqueSortedStringValues {
  param(
    [Parameter(Mandatory = $true)]
    [object]$InputObject
  )

  $valueSet = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)

  if ($null -eq $InputObject) {
    return @()
  }

  if ($InputObject -is [string]) {
    if (-not [string]::IsNullOrWhiteSpace($InputObject)) {
      [void]$valueSet.Add($InputObject)
    }
    $values = @()
    foreach ($value in $valueSet) {
      $values += $value
    }
    return $values | Sort-Object
  }

  if ($InputObject -is [System.Collections.IEnumerable]) {
    foreach ($item in $InputObject) {
      if ($null -eq $item) {
        continue
      }
      $itemString = [string]$item
      if (-not [string]::IsNullOrWhiteSpace($itemString)) {
        [void]$valueSet.Add($itemString)
      }
    }
    $values = @()
    foreach ($value in $valueSet) {
      $values += $value
    }
    return $values | Sort-Object
  }

  $singleValue = [string]$InputObject
  if (-not [string]::IsNullOrWhiteSpace($singleValue)) {
    [void]$valueSet.Add($singleValue)
  }
  $values = @()
  foreach ($value in $valueSet) {
    $values += $value
  }
  return $values | Sort-Object
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
  foreach ($span in (Get-TraceSpans -Trace $Trace)) {
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

  return ,$methods
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
$script:triggerCount = 0
$script:nextTriggerAtSeconds = [double][Math]::Max(1, $TriggerRetryIntervalSeconds)
$triggerStopwatch = [System.Diagnostics.Stopwatch]::StartNew()

Write-Host "Triggering core-agent path via GET $BackendBaseUrl/api/v1/dashboard/summary ..."
Invoke-TraceTriggerRequest -AccessToken $AccessToken -BackendBaseUrl $BackendBaseUrl -RequestId $RequestId
$script:triggerCount++
Write-Host "Dashboard request succeeded. request_id=$RequestId trigger_count=$script:triggerCount"
Write-Host "Polling Jaeger API for correlated trace (timeout=${TraceWaitSeconds}s, retry_interval=${TriggerRetryIntervalSeconds}s) ..."

$script:foundTrace = $null
$script:foundCoreGrpcMethods = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
$script:lastObservation = "No traces fetched yet. trigger_count=0."

try {
  Wait-ObservabilityCondition -Description "Jaeger correlated trace" -TimeoutSeconds $TraceWaitSeconds -TimeoutMessage "No request-correlated cross-service trace found in Jaeger within ${TraceWaitSeconds}s. Check OTEL env vars, collector health, and Jaeger ingestion." -Check {
    if ($triggerStopwatch.Elapsed.TotalSeconds -ge $script:nextTriggerAtSeconds) {
      Invoke-TraceTriggerRequest -AccessToken $AccessToken -BackendBaseUrl $BackendBaseUrl -RequestId $RequestId
      $script:triggerCount++
      $script:nextTriggerAtSeconds += [double][Math]::Max(1, $TriggerRetryIntervalSeconds)
      Write-Host "Re-triggered dashboard request for trace capture. request_id=$RequestId trigger_count=$script:triggerCount"
    }

    $traceCandidates = Get-JaegerTraceCandidates -JaegerBaseUrl $JaegerBaseUrl
    $traces = @($traceCandidates.Traces)
    if ($traces.Count -eq 0) {
      $fallbackSuffix = ""
      if (-not [string]::IsNullOrWhiteSpace($traceCandidates.FallbackError)) {
        $fallbackSuffix = " global_fallback_error=$($traceCandidates.FallbackError)"
      }
      $script:lastObservation = "Jaeger returned 0 traces via $($traceCandidates.Source). trigger_count=$script:triggerCount.$fallbackSuffix"
      return $false
    }

    $validTraceCount = 0
    foreach ($trace in $traces) {
      if ($null -eq $trace -or $null -eq $trace.spans -or $null -eq $trace.processes) {
        continue
      }

      $traceSpans = Get-TraceSpans -Trace $trace
      if ($traceSpans.Count -eq 0) {
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
      foreach ($span in $traceSpans) {
        if ($span.startTime -gt $latestSpanStart) {
          $latestSpanStart = [int64]$span.startTime
        }
      }

      if ($latestSpanStart -lt $requestStartUnixMicros) {
        continue
      }

      if (-not (Trace-HasRequestIdForService -Trace $trace -ServiceName "snowpanel-backend" -RequestId $RequestId)) {
        $script:lastObservation = "Found cross-service trace $($trace.traceID) after request start, but backend span missing snowpanel.request_id=$RequestId."
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
        $observedMethods = (Get-UniqueSortedStringValues -InputObject $coreGrpcMethods) -join ", "
        if ([string]::IsNullOrWhiteSpace($observedMethods)) {
          $observedMethods = "(none)"
        }
        $script:lastObservation = "Found trace $($trace.traceID) with request_id=$RequestId, but missing expected core-agent grpc.method tags: $($missingMethods -join ', '). Observed: $observedMethods"
        continue
      }

      $script:foundTrace = $trace
      $script:foundCoreGrpcMethods = $coreGrpcMethods
      return $true
    }

    if ($validTraceCount -eq 0) {
      $script:lastObservation = "Jaeger returned $($traces.Count) trace envelopes, but none had valid spans/processes. trigger_count=$script:triggerCount."
      return $false
    }

    $script:lastObservation = "Jaeger returned $validTraceCount valid traces, but none matched request_id=$RequestId with required cross-service spans. trigger_count=$script:triggerCount."
    return $false
  }
} catch {
  throw "$($_.Exception.Message) Last observation: $script:lastObservation"
}

$serviceNames = Get-UniqueSortedStringValues -InputObject (Get-TraceServiceSet -Trace $script:foundTrace)
$coreMethods = (Get-UniqueSortedStringValues -InputObject $script:foundCoreGrpcMethods) -join ", "
Write-Host "Trace validation passed."
Write-Host "trace_id: $($script:foundTrace.traceID)"
Write-Host "services: $($serviceNames -join ', ')"
Write-Host "core grpc methods: $coreMethods"
Write-Host "request_id: $RequestId"
Write-Host "trigger_count: $script:triggerCount"
