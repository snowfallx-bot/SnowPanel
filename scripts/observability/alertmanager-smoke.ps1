param(
  [string]$AlertmanagerBaseUrl = "http://127.0.0.1:9093",
  [string]$AlertName = "SnowPanelSmokeAlert",
  [ValidateSet("critical", "warning")]
  [string]$Severity = "critical",
  [string]$ExpectedReceiver = "",
  [string]$Instance = "",
  [int]$AlertDurationSeconds = 120,
  [int]$WaitSeconds = 20
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

if ([string]::IsNullOrWhiteSpace($ExpectedReceiver)) {
  $ExpectedReceiver = Resolve-AlertmanagerReceiver -Severity $Severity
}

if ([string]::IsNullOrWhiteSpace($Instance)) {
  $Instance = "smoke-local-$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
}

$startsAt = [DateTimeOffset]::UtcNow
$endsAt = $startsAt.AddSeconds($AlertDurationSeconds)

$alertPayload = @(
  (New-AlertmanagerSyntheticAlert `
    -AlertName $AlertName `
    -Severity $Severity `
    -Instance $Instance `
    -Source "snowpanel-smoke-script" `
    -Summary "SnowPanel Alertmanager smoke test" `
    -Description "Synthetic alert injected by scripts/observability/alertmanager-smoke.ps1" `
    -StartsAt $startsAt `
    -EndsAt $endsAt)
)

Write-Host "Submitting synthetic alert '$AlertName' to Alertmanager ..."
Submit-AlertmanagerAlerts -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Alerts $alertPayload

$matchLabels = @{
  alertname = $AlertName
  instance  = $Instance
  severity  = $Severity
}

Wait-ObservabilityCondition -Description "Alertmanager routed alert visibility" -TimeoutSeconds $WaitSeconds -TimeoutMessage "Synthetic alert '$AlertName' was not observed with receiver '$ExpectedReceiver' within ${WaitSeconds}s." -Check {
  $alerts = Get-AlertmanagerActiveAlerts -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Labels $matchLabels
  $matchedAlertFound = $false

  foreach ($alert in $alerts) {
    if ($alert.labels.alertname -ne $AlertName -or $alert.labels.instance -ne $Instance) {
      continue
    }

    $matchedAlertFound = $true
    if (Test-AlertmanagerHasReceiver -Alert $alert -ReceiverName $ExpectedReceiver) {
      return $true
    }
  }

  if (-not $matchedAlertFound) {
    return $false
  }

  $groups = Get-AlertmanagerActiveAlertGroups -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Labels $matchLabels
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
