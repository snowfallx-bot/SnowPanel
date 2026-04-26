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

. (Join-Path $PSScriptRoot "common.ps1")

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
Invoke-ObservabilityJsonRequest -Method "POST" -Uri "$AlertmanagerBaseUrl/api/v2/alerts" -Body $alertPayload -ExpectedStatusCodes @(200, 202)

$encodedFilter = [System.Uri]::EscapeDataString("alertname=$AlertName")

Wait-ObservabilityCondition -Description "Alertmanager alert visibility" -TimeoutSeconds $WaitSeconds -TimeoutMessage "Synthetic alert '$AlertName' was not observed in Alertmanager within ${WaitSeconds}s." -Check {
  $alerts = Invoke-ObservabilityJsonRequest -Method "GET" -Uri "$AlertmanagerBaseUrl/api/v2/alerts?active=true&filter=$encodedFilter" -ExpectedStatusCodes @(200)
  foreach ($alert in $alerts) {
    if ($alert.labels.alertname -eq $AlertName -and $alert.labels.instance -eq $Instance) {
      $found = $true
      return $true
    }
  }

  return $false
}

Write-Host "Alertmanager smoke test passed."
Write-Host "alertname: $AlertName"
Write-Host "severity: $Severity"
Write-Host "instance: $Instance"
