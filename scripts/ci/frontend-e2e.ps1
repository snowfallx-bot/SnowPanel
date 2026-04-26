$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

$ProjectName = "snowpanel-e2e"
$BackendPort = "18080"
$FrontendPort = "15173"
$BackendBaseUrl = "http://127.0.0.1:$BackendPort"
$FrontendBaseUrl = "http://127.0.0.1:$FrontendPort"
$BootstrapPassword = "BootstrapSmoke1!"
$RotatedPassword = "BootstrapSmoke2!"
$JwtSecret = "FrontendE2ESecret_2026_Check_123!"
$TestFilesPath = "/tmp"
$ComposeArgs = @("compose", "--project-name", $ProjectName)
$Completed = $false

Assert-DockerAvailable -ScriptPath "scripts/ci/frontend-e2e.ps1"

try {
  $env:APP_ENV = "production"
  $env:BACKEND_PORT = $BackendPort
  $env:FRONTEND_PORT = $FrontendPort
  $env:JWT_SECRET = $JwtSecret
  $env:DEFAULT_ADMIN_PASSWORD = $BootstrapPassword
  $env:LOGIN_ATTEMPT_STORE = "redis"

  Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("up", "-d", "--build", "postgres", "redis", "core-agent", "backend", "frontend")

  Wait-UntilReady -Description "backend readiness" -Check {
    $ready = Invoke-JsonRequest -Method "GET" -Uri "$BackendBaseUrl/ready"
    return $ready.code -eq 0 -and $ready.data.checks.database -eq "up" -and $ready.data.checks.agent -eq "up"
  }

  Wait-UntilReady -Description "frontend startup" -Check {
    $response = Invoke-WebRequest -Uri $FrontendBaseUrl -SkipHttpErrorCheck
    return $response.StatusCode -eq 200
  }

  Wait-UntilReady -Description "frontend proxy health" -Check {
    $proxyHealth = Invoke-JsonRequest -Method "GET" -Uri "$FrontendBaseUrl/health"
    return $proxyHealth.code -eq 0 -and $proxyHealth.data.checks.database -eq "up" -and $proxyHealth.data.checks.agent -eq "up"
  }

  Push-Location frontend
  try {
    $env:PLAYWRIGHT_BASE_URL = $FrontendBaseUrl
    $env:PLAYWRIGHT_API_BASE_URL = $BackendBaseUrl
    $env:PLAYWRIGHT_TEST_FILES_PATH = $TestFilesPath
    $env:PLAYWRIGHT_USERNAME = "admin"
    $env:PLAYWRIGHT_PASSWORD = $BootstrapPassword
    $env:PLAYWRIGHT_ROTATED_PASSWORD = $RotatedPassword

    & npm run test:e2e
    if ($LASTEXITCODE -ne 0) {
      throw "npm run test:e2e failed with exit code $LASTEXITCODE"
    }
  } finally {
    Pop-Location
  }

  $Completed = $true
  Write-Host "Frontend e2e passed."
} finally {
  if (-not $Completed) {
    Write-Host "Frontend e2e failed, printing compose status and logs..."
    try {
      Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("ps")
    } catch {
      Write-Warning $_.Exception.Message
    }
    try {
      Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("logs", "--no-color", "--tail", "200")
    } catch {
      Write-Warning $_.Exception.Message
    }
  }

  try {
    Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("down", "-v", "--remove-orphans")
  } catch {
    Write-Warning $_.Exception.Message
  }
}
