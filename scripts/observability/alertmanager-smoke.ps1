param(
  [string]$AlertmanagerBaseUrl = "http://127.0.0.1:9093",
  [string]$AlertName = "SnowPanelSmokeAlert",
  [ValidateSet("critical", "warning")]
  [string]$Severity = "critical",
  [string]$ExpectedReceiver = "",
  [string]$Instance = "smoke-local",
  [int]$AlertDurationSeconds = 120,
  [int]$WaitSeconds = 20
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

if ([string]::IsNullOrWhiteSpace($ExpectedReceiver)) {
  switch ($Severity) {
    "critical" { $ExpectedReceiver = "snowpanel-critical" }
    "warning" { $ExpectedReceiver = "snowpanel-warning" }
    default { throw "Unsupported severity '$Severity' for default receiver resolution." }
  }
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
Invoke-ObservabilityJsonRequest -Method "POST" -Uri "$AlertmanagerBaseUrl/api/v2/alerts" -Body $alertPayload -ExpectedStatusCodes @(200, 202)

$encodedFilter = [System.Uri]::EscapeDataString("alertname=$AlertName")

Wait-ObservabilityCondition -Description "Alertmanager routed alert visibility" -TimeoutSeconds $WaitSeconds -TimeoutMessage "Synthetic alert '$AlertName' was not observed with receiver '$ExpectedReceiver' within ${WaitSeconds}s." -Check {
  $alerts = Invoke-ObservabilityJsonRequest -Method "GET" -Uri "$AlertmanagerBaseUrl/api/v2/alerts?active=true&filter=$encodedFilter" -ExpectedStatusCodes @(200)
  $matchedAlertFound = $false

  foreach ($alert in $alerts) {
    if ($alert.labels.alertname -ne $AlertName -or $alert.labels.instance -ne $Instance) {
      continue
    }

    $matchedAlertFound = $true
    if ($null -eq $alert.receivers) {
      continue
    }

    foreach ($receiver in $alert.receivers) {
      $receiverName = if ($receiver -is [string]) { [string]$receiver } else { [string]$receiver.name }
      if (-not [string]::IsNullOrWhiteSpace($receiverName) -and $receiverName -ieq $ExpectedReceiver) {
        return $true
      }
    }
  }

  if (-not $matchedAlertFound) {
    return $false
  }

  $groups = Invoke-ObservabilityJsonRequest -Method "GET" -Uri "$AlertmanagerBaseUrl/api/v2/alerts/groups?active=true&filter=$encodedFilter" -ExpectedStatusCodes @(200)
  foreach ($group in $groups) {
    $groupReceiverName = [string]$group.receiver.name
    if ([string]::IsNullOrWhiteSpace($groupReceiverName) -or $groupReceiverName -ine $ExpectedReceiver) {
      continue
    }

    foreach ($groupAlert in $group.alerts) {
      if ($groupAlert.labels.alertname -eq $AlertName -and $groupAlert.labels.instance -eq $Instance) {
        return $true
      }
    }
  }

  return $false
}

Write-Host "Alertmanager smoke test passed."
Write-Host "alertname: $AlertName"
Write-Host "severity: $Severity"
Write-Host "instance: $Instance"
Write-Host "receiver: $ExpectedReceiver"
