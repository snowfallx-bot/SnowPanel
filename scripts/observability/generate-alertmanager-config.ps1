param(
  [Parameter(Mandatory = $true)]
  [string]$WarningWebhookUrl,
  [Parameter(Mandatory = $true)]
  [string]$CriticalWebhookUrl,
  [string]$CriticalEscalationWebhookUrl = "",
  [string]$OutputPath = "deploy/observability/alertmanager/alertmanager.generated.yml",
  [string]$WarningRepeatInterval = "4h",
  [string]$CriticalRepeatInterval = "30m",
  [string]$CriticalEscalationRepeatInterval = "15m"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Assert-ValidWebhookUrl {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Name,
    [Parameter(Mandatory = $true)]
    [string]$Value
  )

  if ([string]::IsNullOrWhiteSpace($Value)) {
    throw "$Name must not be empty."
  }

  $uri = $null
  if (-not [System.Uri]::TryCreate($Value, [System.UriKind]::Absolute, [ref]$uri)) {
    throw "$Name is not a valid absolute URL: $Value"
  }

  if ($uri.Scheme -notin @("https", "http")) {
    throw "$Name must use http/https URL scheme: $Value"
  }
}

Assert-ValidWebhookUrl -Name "WarningWebhookUrl" -Value $WarningWebhookUrl
Assert-ValidWebhookUrl -Name "CriticalWebhookUrl" -Value $CriticalWebhookUrl

$hasEscalation = -not [string]::IsNullOrWhiteSpace($CriticalEscalationWebhookUrl)
if ($hasEscalation) {
  Assert-ValidWebhookUrl -Name "CriticalEscalationWebhookUrl" -Value $CriticalEscalationWebhookUrl
}

$resolvedOutputPath = $OutputPath
if (-not [System.IO.Path]::IsPathRooted($resolvedOutputPath)) {
  $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
  $resolvedOutputPath = Join-Path $repoRoot $resolvedOutputPath
}

$outputDir = Split-Path -Path $resolvedOutputPath -Parent
if (-not [string]::IsNullOrWhiteSpace($outputDir)) {
  New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
}

$yamlLines = @(
  "global:"
  "  resolve_timeout: 5m"
  ""
  "route:"
  "  receiver: snowpanel-warning"
  "  group_by: [""alertname"", ""severity"", ""instance""]"
  "  group_wait: 30s"
  "  group_interval: 10m"
  "  repeat_interval: $WarningRepeatInterval"
  "  routes:"
  "    - receiver: snowpanel-critical"
  "      matchers:"
  "        - severity=""critical"""
  "      group_wait: 10s"
  "      group_interval: 2m"
  "      repeat_interval: $CriticalRepeatInterval"
)

if ($hasEscalation) {
  $yamlLines += @(
    "      continue: true"
    "    - receiver: snowpanel-critical-escalation"
    "      matchers:"
    "        - severity=""critical"""
    "      group_wait: 2m"
    "      group_interval: 10m"
    "      repeat_interval: $CriticalEscalationRepeatInterval"
  )
}

$yamlLines += @(
  "    - receiver: snowpanel-warning"
  "      matchers:"
  "        - severity=""warning"""
  "      group_wait: 30s"
  "      group_interval: 10m"
  "      repeat_interval: $WarningRepeatInterval"
  ""
  "receivers:"
  "  - name: snowpanel-warning"
  "    webhook_configs:"
  "      - url: ""$WarningWebhookUrl"""
  "        send_resolved: true"
  ""
  "  - name: snowpanel-critical"
  "    webhook_configs:"
  "      - url: ""$CriticalWebhookUrl"""
  "        send_resolved: true"
)

if ($hasEscalation) {
  $yamlLines += @(
    ""
    "  - name: snowpanel-critical-escalation"
    "    webhook_configs:"
    "      - url: ""$CriticalEscalationWebhookUrl"""
    "        send_resolved: true"
  )
}

$yamlLines += @(
  ""
  "inhibit_rules:"
  "  - source_matchers:"
  "      - severity=""critical"""
  "    target_matchers:"
  "      - severity=""warning"""
  "    equal: [""alertname"", ""instance""]"
)

$content = [string]::Join([Environment]::NewLine, $yamlLines) + [Environment]::NewLine
Set-Content -Path $resolvedOutputPath -Value $content -Encoding UTF8

Write-Host "Generated Alertmanager config: $resolvedOutputPath"
Write-Host "warning webhook: $WarningWebhookUrl"
Write-Host "critical webhook: $CriticalWebhookUrl"
if ($hasEscalation) {
  Write-Host "critical escalation webhook: $CriticalEscalationWebhookUrl"
}
