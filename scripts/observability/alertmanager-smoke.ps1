param(
  [string]$AlertmanagerBaseUrl = "http://127.0.0.1:9093",
  [string]$AlertName = "SnowPanelSmokeAlert",
  [ValidateSet("critical", "warning")]
  [string]$Severity = "critical",
  [string]$ExpectedReceiver = "",
  [string]$Instance = "",
  [int]$AlertDurationSeconds = 120,
  [ValidateRange(10, 300)]
  [int]$WaitSeconds = 60
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

if ([string]::IsNullOrWhiteSpace($ExpectedReceiver)) {
  $ExpectedReceiver = Resolve-AlertmanagerReceiver -Severity $Severity
}

if ([string]::IsNullOrWhiteSpace($Instance)) {
  $Instance = New-ObservabilityInstanceId -Prefix "smoke-local"
}

$timeWindow = New-AlertTimeWindow -DurationSeconds $AlertDurationSeconds
$startsAt = $timeWindow.StartsAt
$endsAt = $timeWindow.EndsAt

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
  $matchIdentityLabels = @{
    alertname = $AlertName
    instance  = $Instance
  }

  foreach ($alert in @($alerts)) {
    if (Test-AlertmanagerLabelsMatch -Alert $alert -Labels $matchIdentityLabels) {
      if (Test-AlertmanagerHasReceiver -Alert $alert -ReceiverName $ExpectedReceiver) {
        return $true
      }
    }
  }

  $groups = Get-AlertmanagerActiveAlertGroups -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Labels $matchLabels
  foreach ($group in $groups) {
    $groupReceiverName = Get-AlertmanagerReceiverName -Receiver $group.receiver
    if ([string]::IsNullOrWhiteSpace($groupReceiverName) -or $groupReceiverName -ine $ExpectedReceiver) {
      continue
    }

    $groupAlert = Find-AlertmanagerAlertByLabels -Alerts $group.alerts -Labels $matchIdentityLabels
    if ($null -ne $groupAlert) {
      return $true
    }
  }

  return $false
}

Write-Host "Alertmanager smoke test passed."
Write-Host "alertname: $AlertName"
Write-Host "severity: $Severity"
Write-Host "instance: $Instance"
Write-Host "receiver: $ExpectedReceiver"
