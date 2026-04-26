param(
  [string]$PrometheusImage = "prom/prometheus:v2.54.1",
  [string]$AlertmanagerImage = "prom/alertmanager:v0.28.1"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$prometheusDir = Join-Path $repoRoot "deploy\observability\prometheus"
$alertmanagerDir = Join-Path $repoRoot "deploy\observability\alertmanager"
$prometheusConfigFileHostPath = Join-Path $prometheusDir "prometheus.yml"
$prometheusRuleDirectory = Join-Path $prometheusDir "alerts"
$prometheusRuleTestFileHostPath = Join-Path $prometheusDir "tests\snowpanel-alerts.test.yml"
$alertmanagerBaselineConfigHostPath = Join-Path $alertmanagerDir "alertmanager.yml"
$alertmanagerProductionConfigHostPath = Join-Path $alertmanagerDir "alertmanager.production.example.yml"
$prometheusConfigFileContainerPath = "/etc/prometheus/prometheus.yml"
$prometheusRuleTestFileContainerPath = "/etc/prometheus/tests/snowpanel-alerts.test.yml"
$alertmanagerBaselineConfigContainerPath = "/etc/alertmanager/alertmanager.yml"
$alertmanagerProductionConfigContainerPath = "/etc/alertmanager/alertmanager.production.example.yml"

if (-not (Test-Path -LiteralPath $prometheusRuleTestFileHostPath)) {
  throw "Prometheus alert rule test file not found: $prometheusRuleTestFileHostPath"
}

$prometheusRuleFilesHostPaths = @(
  Get-ChildItem -Path $prometheusRuleDirectory -Filter "*.yml" -File |
    Sort-Object -Property Name |
    ForEach-Object { $_.FullName }
)
if ($prometheusRuleFilesHostPaths.Count -eq 0) {
  throw "No Prometheus rule files found in $prometheusRuleDirectory"
}

$prometheusRuleFilesContainerPaths = @(
  $prometheusRuleFilesHostPaths |
    ForEach-Object { "/etc/prometheus/alerts/$([System.IO.Path]::GetFileName($_))" }
)

$dockerAvailable = $null -ne (Get-Command docker -ErrorAction SilentlyContinue)
$promtoolAvailable = $null -ne (Get-Command promtool -ErrorAction SilentlyContinue)
$amtoolAvailable = $null -ne (Get-Command amtool -ErrorAction SilentlyContinue)
$useDocker = $dockerAvailable

if (-not $useDocker) {
  if ($promtoolAvailable -and $amtoolAvailable) {
    Write-Host "docker command not found. Falling back to local promtool/amtool binaries."
  } else {
    throw "docker command not found, and local promtool/amtool binaries are unavailable. Install Docker or provide promtool + amtool in PATH."
  }
}

function Invoke-ExternalCommand {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Executable,

    [Parameter(Mandatory = $true)]
    [string[]]$Arguments
  )

  & $Executable @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "$Executable $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
  }
}

function Invoke-Promtool {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$LocalArguments,

    [Parameter(Mandatory = $true)]
    [string[]]$ContainerArguments
  )

  if ($useDocker) {
    Invoke-ExternalCommand -Executable "docker" -Arguments (
      @("run", "--rm", "--entrypoint", "promtool", "-v", "${prometheusDir}:/etc/prometheus:ro", $PrometheusImage) + $ContainerArguments
    )
    return
  }

  Invoke-ExternalCommand -Executable "promtool" -Arguments $LocalArguments
}

function Invoke-Amtool {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$LocalArguments,

    [Parameter(Mandatory = $true)]
    [string[]]$ContainerArguments
  )

  if ($useDocker) {
    Invoke-ExternalCommand -Executable "docker" -Arguments (
      @("run", "--rm", "--entrypoint", "amtool", "-v", "${alertmanagerDir}:/etc/alertmanager:ro", $AlertmanagerImage) + $ContainerArguments
    )
    return
  }

  Invoke-ExternalCommand -Executable "amtool" -Arguments $LocalArguments
}

Write-Host "Validating Prometheus config ..."
Invoke-Promtool -LocalArguments @(
  "check", "config", $prometheusConfigFileHostPath
) -ContainerArguments @(
  "check", "config", $prometheusConfigFileContainerPath
)

Write-Host "Validating Prometheus alert/rule files ..."
Invoke-Promtool -LocalArguments (
  @("check", "rules") + $prometheusRuleFilesHostPaths
) -ContainerArguments (
  @("check", "rules") + $prometheusRuleFilesContainerPaths
)

Write-Host "Running Prometheus alert rule unit tests ..."
Invoke-Promtool -LocalArguments @(
  "test", "rules", $prometheusRuleTestFileHostPath
) -ContainerArguments @(
  "test", "rules", $prometheusRuleTestFileContainerPath
)

Write-Host "Validating Alertmanager baseline config ..."
Invoke-Amtool -LocalArguments @(
  "check-config", $alertmanagerBaselineConfigHostPath
) -ContainerArguments @(
  "check-config", $alertmanagerBaselineConfigContainerPath
)

Write-Host "Validating Alertmanager production example config ..."
Invoke-Amtool -LocalArguments @(
  "check-config", $alertmanagerProductionConfigHostPath
) -ContainerArguments @(
  "check-config", $alertmanagerProductionConfigContainerPath
)

Write-Host "Observability config validation passed."
