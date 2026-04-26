param(
  [string]$HostAgentPid = ""
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$pidText = $HostAgentPid
if ([string]::IsNullOrWhiteSpace($pidText)) {
  $pidText = $env:HOST_AGENT_PID
}

if ([string]::IsNullOrWhiteSpace($pidText)) {
  Write-Host "HOST_AGENT_PID is empty; skipping host core-agent shutdown."
  exit 0
}

$pidValue = 0
if (-not [int]::TryParse($pidText, [ref]$pidValue) -or $pidValue -le 0) {
  throw "Invalid host-agent PID '$pidText'."
}

try {
  Stop-Process -Id $pidValue -Force -ErrorAction Stop
  Write-Host "Stopped host core-agent process PID $pidValue."
} catch {
  if ($_.Exception.Message -match "Cannot find a process with the process identifier") {
    Write-Warning "Host core-agent process PID $pidValue was not running."
    exit 0
  }
  throw
}
