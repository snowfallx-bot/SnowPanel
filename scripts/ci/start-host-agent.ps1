param(
  [string]$HostAgentTarget = "host.docker.internal:50051",
  [string]$HostAgentMetricsBaseUrl = "http://127.0.0.1:9108",
  [string]$CoreAgentBinaryPath = "core-agent/target/release/core-agent",
  [string]$LogDir = ".github/workflow-logs",
  [string]$OtlpEndpoint = "127.0.0.1:4317",
  [int]$WaitTimeoutSeconds = 20,
  [int]$WaitIntervalMilliseconds = 500,
  [string]$GithubEnvFile = ""
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if ([string]::IsNullOrWhiteSpace($HostAgentTarget)) {
  throw "HostAgentTarget must not be empty."
}

$targetSegments = $HostAgentTarget.Split(":")
if ($targetSegments.Count -lt 2) {
  throw "HostAgentTarget '$HostAgentTarget' must be in '<host>:<port>' format."
}

$agentPort = 0
if (-not [int]::TryParse($targetSegments[-1], [ref]$agentPort) -or $agentPort -lt 1 -or $agentPort -gt 65535) {
  throw "HostAgentTarget '$HostAgentTarget' has invalid port."
}

if ([string]::IsNullOrWhiteSpace($HostAgentMetricsBaseUrl)) {
  throw "HostAgentMetricsBaseUrl must not be empty."
}

$metricsUri = $null
if (-not [Uri]::TryCreate($HostAgentMetricsBaseUrl, [System.UriKind]::Absolute, [ref]$metricsUri)) {
  throw "HostAgentMetricsBaseUrl '$HostAgentMetricsBaseUrl' is not a valid absolute URI."
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$agentBinary = $CoreAgentBinaryPath
if (-not [System.IO.Path]::IsPathRooted($agentBinary)) {
  $agentBinary = Join-Path $repoRoot $agentBinary
}
if (-not (Test-Path -LiteralPath $agentBinary)) {
  throw "core-agent binary not found at $agentBinary"
}

$resolvedLogDir = $LogDir
if (-not [System.IO.Path]::IsPathRooted($resolvedLogDir)) {
  $resolvedLogDir = Join-Path $repoRoot $resolvedLogDir
}
New-Item -ItemType Directory -Force -Path $resolvedLogDir | Out-Null

$stdoutLog = Join-Path $resolvedLogDir "host-agent.stdout.log"
$stderrLog = Join-Path $resolvedLogDir "host-agent.stderr.log"

$agentEnv = @{
  APP_ENV                     = "production"
  CORE_AGENT_HOST             = "0.0.0.0"
  CORE_AGENT_PORT             = "$agentPort"
  CORE_AGENT_METRICS_ENABLED  = "true"
  CORE_AGENT_METRICS_HOST     = "127.0.0.1"
  CORE_AGENT_METRICS_PORT     = "$($metricsUri.Port)"
  OTEL_TRACING_ENABLED        = "true"
  OTEL_SERVICE_NAME           = "snowpanel-core-agent"
  OTEL_EXPORTER_OTLP_ENDPOINT = $OtlpEndpoint
  OTEL_EXPORTER_OTLP_INSECURE = "true"
  OTEL_TRACES_SAMPLER_ARG     = "1.0"
}

foreach ($pair in $agentEnv.GetEnumerator()) {
  [System.Environment]::SetEnvironmentVariable($pair.Key, $pair.Value, "Process")
}

$proc = Start-Process -FilePath $agentBinary -WorkingDirectory $repoRoot -RedirectStandardOutput $stdoutLog -RedirectStandardError $stderrLog -PassThru
Write-Host "Started host core-agent process with PID $($proc.Id)."

$started = $false
$attempts = [math]::Ceiling(($WaitTimeoutSeconds * 1000) / [double]$WaitIntervalMilliseconds)
for ($attempt = 1; $attempt -le $attempts; $attempt++) {
  $proc.Refresh()
  if ($proc.HasExited) {
    throw "core-agent exited unexpectedly with code $($proc.ExitCode)."
  }

  $client = [System.Net.Sockets.TcpClient]::new()
  try {
    $task = $client.ConnectAsync("127.0.0.1", $agentPort)
    if ($task.Wait($WaitIntervalMilliseconds) -and $client.Connected) {
      $started = $true
      break
    }
  } catch {
  } finally {
    $client.Dispose()
  }

  Start-Sleep -Milliseconds $WaitIntervalMilliseconds
}

if (-not $started) {
  throw "core-agent did not bind to 127.0.0.1:$agentPort within timeout."
}

$targetEnvFile = $GithubEnvFile
if ([string]::IsNullOrWhiteSpace($targetEnvFile) -and -not [string]::IsNullOrWhiteSpace($env:GITHUB_ENV)) {
  $targetEnvFile = $env:GITHUB_ENV
}
if (-not [string]::IsNullOrWhiteSpace($targetEnvFile)) {
  "HOST_AGENT_PID=$($proc.Id)" | Out-File -FilePath $targetEnvFile -Encoding utf8 -Append
}

Write-Output "$($proc.Id)"
