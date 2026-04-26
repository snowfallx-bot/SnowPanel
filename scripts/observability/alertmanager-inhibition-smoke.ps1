param(
  [string]$AlertmanagerBaseUrl = "http://127.0.0.1:9093",
  [string]$AlertName = "SnowPanelInhibitionSmokeAlert",
  [string]$Instance = "",
  [int]$AlertDurationSeconds = 180,
  [int]$WaitSeconds = 30
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

if ([string]::IsNullOrWhiteSpace($Instance)) {
  $Instance = "smoke-inhibit-$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
}

$startsAt = [DateTimeOffset]::UtcNow
$endsAt = $startsAt.AddSeconds($AlertDurationSeconds)
$warningReceiver = Resolve-AlertmanagerReceiver -Severity "warning"
$criticalReceiver = Resolve-AlertmanagerReceiver -Severity "critical"

$baseLabels = @{
  alertname = $AlertName
  instance  = $Instance
}

Write-Host "Submitting warning alert '$AlertName' to Alertmanager ..."
Submit-AlertmanagerAlerts -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Alerts @(
  (New-AlertmanagerSyntheticAlert `
    -AlertName $AlertName `
    -Severity "warning" `
    -Instance $Instance `
    -Source "snowpanel-inhibition-smoke" `
    -Summary "SnowPanel Alertmanager inhibition smoke test" `
    -Description "Synthetic warning alert injected by scripts/observability/alertmanager-inhibition-smoke.ps1" `
    -StartsAt $startsAt `
    -EndsAt $endsAt)
)

Wait-ObservabilityCondition -Description "warning alert routing visibility" -TimeoutSeconds $WaitSeconds -TimeoutMessage "Warning alert '$AlertName' was not routed to $warningReceiver within ${WaitSeconds}s." -Check {
  $warningLabels = @{
    alertname = $AlertName
    instance  = $Instance
    severity  = "warning"
  }
  $alerts = Get-AlertmanagerActiveAlerts -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Labels $warningLabels
  foreach ($alert in $alerts) {
    if (-not (Test-AlertmanagerLabelsMatch -Alert $alert -Labels $warningLabels)) {
      continue
    }
    return (Test-AlertmanagerHasReceiver -Alert $alert -ReceiverName $warningReceiver)
  }
  return $false
}

Write-Host "Submitting critical alert '$AlertName' to trigger inhibition ..."
Submit-AlertmanagerAlerts -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Alerts @(
  (New-AlertmanagerSyntheticAlert `
    -AlertName $AlertName `
    -Severity "critical" `
    -Instance $Instance `
    -Source "snowpanel-inhibition-smoke" `
    -Summary "SnowPanel Alertmanager inhibition smoke test" `
    -Description "Synthetic critical alert injected by scripts/observability/alertmanager-inhibition-smoke.ps1" `
    -StartsAt $startsAt `
    -EndsAt $endsAt)
)

Wait-ObservabilityCondition -Description "warning inhibited by critical" -TimeoutSeconds $WaitSeconds -TimeoutMessage "Critical/warning inhibition did not converge for '$AlertName' within ${WaitSeconds}s." -Check {
  $alerts = Get-AlertmanagerActiveAlerts -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Labels $baseLabels
  $warningAlert = $null
  $criticalAlert = $null
  $warningLabels = @{
    alertname = $AlertName
    instance  = $Instance
    severity  = "warning"
  }
  $criticalLabels = @{
    alertname = $AlertName
    instance  = $Instance
    severity  = "critical"
  }

  foreach ($alert in $alerts) {
    if (Test-AlertmanagerLabelsMatch -Alert $alert -Labels $warningLabels) {
      $warningAlert = $alert
      continue
    }
    if (Test-AlertmanagerLabelsMatch -Alert $alert -Labels $criticalLabels) {
      $criticalAlert = $alert
    }
  }

  if ($null -eq $warningAlert -or $null -eq $criticalAlert) {
    return $false
  }

  if (-not (Test-AlertmanagerHasReceiver -Alert $criticalAlert -ReceiverName $criticalReceiver)) {
    return $false
  }

  $inhibitedByCount = 0
  if ($null -ne $warningAlert.status -and $null -ne $warningAlert.status.inhibitedBy) {
    $inhibitedByCount = @($warningAlert.status.inhibitedBy).Count
  }

  return $inhibitedByCount -gt 0
}

Write-Host "Alertmanager inhibition smoke test passed."
Write-Host "alertname: $AlertName"
Write-Host "instance: $Instance"
