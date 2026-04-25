param(
  [string]$AlertmanagerBaseUrl = "http://127.0.0.1:9093",
  [string]$AlertName = "SnowPanelSmokeAlert",
  [ValidateSet("critical", "warning")]
  [string]$Severity = "critical",
  [string]$Instance = "smoke-local",
  [int]$AlertDurationSeconds = 120,
  [int]$WaitSeconds = 20
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest
$SupportsSkipHttpErrorCheck = (Get-Command Invoke-WebRequest).Parameters.ContainsKey("SkipHttpErrorCheck")

function Invoke-JsonRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Method,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [object]$Body = $null,
    [int[]]$ExpectedStatusCodes = @(200)
  )

  $requestParams = @{
    Method  = $Method
    Uri     = $Uri
    Headers = @{
      "Content-Type" = "application/json"
    }
  }

  if ($SupportsSkipHttpErrorCheck) {
    $requestParams.SkipHttpErrorCheck = $true
  }

  if ($null -ne $Body) {
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

$startsAt = [DateTimeOffset]::UtcNow
$endsAt = $startsAt.AddSeconds($AlertDurationSeconds)

$alertPayload = @(
  @{
    labels = @{
      alertname = $AlertName
      severity  = $Severity
      instance  = $Instance
      source    = "snowpanel-smoke-script"
    }
    annotations = @{
      summary     = "SnowPanel Alertmanager smoke test"
      description = "Synthetic alert injected by scripts/observability/alertmanager-smoke.ps1"
    }
    startsAt = $startsAt.ToString("o")
    endsAt   = $endsAt.ToString("o")
  }
)

Write-Host "Submitting synthetic alert '$AlertName' to Alertmanager ..."
Invoke-JsonRequest -Method "POST" -Uri "$AlertmanagerBaseUrl/api/v2/alerts" -Body $alertPayload -ExpectedStatusCodes @(200, 202)

$deadline = (Get-Date).AddSeconds($WaitSeconds)
$found = $false
$encodedFilter = [System.Uri]::EscapeDataString("alertname=$AlertName")

while ((Get-Date) -lt $deadline) {
  $alerts = Invoke-JsonRequest -Method "GET" -Uri "$AlertmanagerBaseUrl/api/v2/alerts?active=true&filter=$encodedFilter" -ExpectedStatusCodes @(200)
  foreach ($alert in $alerts) {
    if ($alert.labels.alertname -eq $AlertName -and $alert.labels.instance -eq $Instance) {
      $found = $true
      break
    }
  }

  if ($found) {
    break
  }

  Start-Sleep -Seconds 2
}

if (-not $found) {
  throw "Synthetic alert '$AlertName' was not observed in Alertmanager within ${WaitSeconds}s."
}

Write-Host "Alertmanager smoke test passed."
Write-Host "alertname: $AlertName"
Write-Host "severity: $Severity"
Write-Host "instance: $Instance"
