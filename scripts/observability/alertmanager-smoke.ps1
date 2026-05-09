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

Wait-ObservabilityCondition -Description "Alertmanager routed alert visibility" -TimeoutSeconds $WaitSeconds -TimeoutMessage "Synthetic alert '$AlertName' was not observed with receiver '$ExpectedReceiver' within ${WaitSeconds}s." -Check {
  $alerts = Get-AlertmanagerActiveAlerts -AlertmanagerBaseUrl $AlertmanagerBaseUrl
  $matchIdentityLabels = @{
    alertname = $AlertName
    instance  = $Instance
  }

  $matchedAlert = Find-AlertmanagerAlertByLabels -Alerts $alerts -Labels $matchIdentityLabels
  $matchedAlertReceiverNames = @()
  if ($null -ne $matchedAlert) {
    if (Test-AlertmanagerHasReceiver -Alert $matchedAlert -ReceiverName $ExpectedReceiver) {
      return $true
    }

    $matchedAlertReceiverNames = Get-AlertmanagerReceiverNames -Alert $matchedAlert
  }

  $groupReceiverEvidenceSeen = $false
  foreach ($alert in @($alerts)) {
    if (Test-AlertmanagerLabelsMatch -Alert $alert -Labels $matchIdentityLabels) {
      if (Test-AlertmanagerHasReceiver -Alert $alert -ReceiverName $ExpectedReceiver) {
        return $true
      }
    }
  }

  $groups = Get-AlertmanagerActiveAlertGroups -AlertmanagerBaseUrl $AlertmanagerBaseUrl
  foreach ($group in $groups) {
    $groupReceiverName = Get-AlertmanagerReceiverName -Receiver $group.receiver
    if (-not [string]::IsNullOrWhiteSpace($groupReceiverName)) {
      $groupReceiverEvidenceSeen = $true
    }
    if ([string]::IsNullOrWhiteSpace($groupReceiverName) -or $groupReceiverName -ine $ExpectedReceiver) {
      continue
    }

    $groupAlert = Find-AlertmanagerAlertByLabels -Alerts $group.alerts -Labels $matchIdentityLabels
    if ($null -ne $groupAlert) {
      return $true
    }
  }

  if ($null -ne $matchedAlert -and @($matchedAlertReceiverNames).Count -eq 0 -and -not $groupReceiverEvidenceSeen) {
    Write-Warning "Alertmanager API did not expose receiver fields for alert '$AlertName'; falling back to alert visibility check."
    return $true
  }

  return $false
}

Write-Host "Alertmanager smoke test passed."
Write-Host "alertname: $AlertName"
Write-Host "severity: $Severity"
Write-Host "instance: $Instance"
Write-Host "receiver: $ExpectedReceiver"
