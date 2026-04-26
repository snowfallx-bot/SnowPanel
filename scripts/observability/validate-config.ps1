param(
  [string]$PrometheusImage = "prom/prometheus:v2.54.1",
  [string]$AlertmanagerImage = "prom/alertmanager:v0.28.1"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  throw "docker command not found. Install Docker/Compose before running scripts/observability/validate-config.ps1."
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$prometheusDir = Join-Path $repoRoot "deploy\observability\prometheus"
$alertmanagerDir = Join-Path $repoRoot "deploy\observability\alertmanager"
$prometheusRuleTestFile = "/etc/prometheus/tests/snowpanel-alerts.test.yml"

function Invoke-Docker {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$Arguments
  )

  & docker @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "docker $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
  }
}

Write-Host "Validating Prometheus config ..."
Invoke-Docker -Arguments @(
  "run", "--rm",
  "-v", "${prometheusDir}:/etc/prometheus:ro",
  $PrometheusImage,
  "promtool", "check", "config", "/etc/prometheus/prometheus.yml"
)

Write-Host "Validating Prometheus alert/rule files ..."
Invoke-Docker -Arguments @(
  "run", "--rm",
  "-v", "${prometheusDir}:/etc/prometheus:ro",
  $PrometheusImage,
  "/bin/sh", "-lc", "promtool check rules /etc/prometheus/alerts/*.yml"
)

Write-Host "Running Prometheus alert rule unit tests ..."
Invoke-Docker -Arguments @(
  "run", "--rm",
  "-v", "${prometheusDir}:/etc/prometheus:ro",
  $PrometheusImage,
  "promtool", "test", "rules", $prometheusRuleTestFile
)

Write-Host "Validating Alertmanager baseline config ..."
Invoke-Docker -Arguments @(
  "run", "--rm",
  "-v", "${alertmanagerDir}:/etc/alertmanager:ro",
  $AlertmanagerImage,
  "amtool", "check-config", "/etc/alertmanager/alertmanager.yml"
)

Write-Host "Validating Alertmanager production example config ..."
Invoke-Docker -Arguments @(
  "run", "--rm",
  "-v", "${alertmanagerDir}:/etc/alertmanager:ro",
  $AlertmanagerImage,
  "amtool", "check-config", "/etc/alertmanager/alertmanager.production.example.yml"
)

Write-Host "Observability config validation passed."
