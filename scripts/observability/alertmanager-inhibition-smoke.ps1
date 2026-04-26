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

$baseLabels = @{
  alertname = $AlertName
  instance  = $Instance
}

Write-Host "Submitting warning alert '$AlertName' to Alertmanager ..."
Invoke-ObservabilityJsonRequest -Method "POST" -Uri "$AlertmanagerBaseUrl/api/v2/alerts" -Body @(
  (New-AlertmanagerSyntheticAlert `
    -AlertName $AlertName `
    -Severity "warning" `
    -Instance $Instance `
    -Source "snowpanel-inhibition-smoke" `
    -Summary "SnowPanel Alertmanager inhibition smoke test" `
    -Description "Synthetic warning alert injected by scripts/observability/alertmanager-inhibition-smoke.ps1" `
    -StartsAt $startsAt `
    -EndsAt $endsAt)
) -ExpectedStatusCodes @(200, 202)

Wait-ObservabilityCondition -Description "warning alert routing visibility" -TimeoutSeconds $WaitSeconds -TimeoutMessage "Warning alert '$AlertName' was not routed to snowpanel-warning within ${WaitSeconds}s." -Check {
  $warningLabels = @{
    alertname = $AlertName
    instance  = $Instance
    severity  = "warning"
  }
  $alerts = Get-AlertmanagerActiveAlerts -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Labels $warningLabels
  foreach ($alert in $alerts) {
    if ($alert.labels.alertname -ne $AlertName -or $alert.labels.instance -ne $Instance -or $alert.labels.severity -ne "warning") {
      continue
    }
    return (Test-AlertmanagerHasReceiver -Alert $alert -ReceiverName "snowpanel-warning")
  }
  return $false
}

Write-Host "Submitting critical alert '$AlertName' to trigger inhibition ..."
Invoke-ObservabilityJsonRequest -Method "POST" -Uri "$AlertmanagerBaseUrl/api/v2/alerts" -Body @(
  (New-AlertmanagerSyntheticAlert `
    -AlertName $AlertName `
    -Severity "critical" `
    -Instance $Instance `
    -Source "snowpanel-inhibition-smoke" `
    -Summary "SnowPanel Alertmanager inhibition smoke test" `
    -Description "Synthetic critical alert injected by scripts/observability/alertmanager-inhibition-smoke.ps1" `
    -StartsAt $startsAt `
    -EndsAt $endsAt)
) -ExpectedStatusCodes @(200, 202)

Wait-ObservabilityCondition -Description "warning inhibited by critical" -TimeoutSeconds $WaitSeconds -TimeoutMessage "Critical/warning inhibition did not converge for '$AlertName' within ${WaitSeconds}s." -Check {
  $alerts = Get-AlertmanagerActiveAlerts -AlertmanagerBaseUrl $AlertmanagerBaseUrl -Labels $baseLabels
  $warningAlert = $null
  $criticalAlert = $null

  foreach ($alert in $alerts) {
    if ($alert.labels.alertname -ne $AlertName -or $alert.labels.instance -ne $Instance) {
      continue
    }

    switch ([string]$alert.labels.severity) {
      "warning" { $warningAlert = $alert }
      "critical" { $criticalAlert = $alert }
    }
  }

  if ($null -eq $warningAlert -or $null -eq $criticalAlert) {
    return $false
  }

  if (-not (Test-AlertmanagerHasReceiver -Alert $criticalAlert -ReceiverName "snowpanel-critical")) {
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
