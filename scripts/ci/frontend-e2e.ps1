$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

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

function Invoke-Compose {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$Arguments
  )

  & docker @ComposeArgs @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "docker compose $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
  }
}

function Invoke-JsonRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Method,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [object]$Body = $null,
    [hashtable]$Headers = @{},
    [int[]]$ExpectedStatusCodes = @(200)
  )

  $requestParams = @{
    Method             = $Method
    Uri                = $Uri
    Headers            = $Headers
    SkipHttpErrorCheck = $true
  }

  if ($null -ne $Body) {
    $requestParams.ContentType = "application/json"
    $requestParams.Body = ($Body | ConvertTo-Json -Depth 10 -Compress)
  }

  $response = Invoke-WebRequest @requestParams
  if ($response.StatusCode -notin $ExpectedStatusCodes) {
    throw "Expected status $($ExpectedStatusCodes -join ', ') from $Method $Uri, got $($response.StatusCode). Body: $($response.Content)"
  }

  if ([string]::IsNullOrWhiteSpace($response.Content)) {
    return $null
  }

  return $response.Content | ConvertFrom-Json -Depth 10
}

function Wait-UntilReady {
  param(
    [Parameter(Mandatory = $true)]
    [scriptblock]$Check,
    [Parameter(Mandatory = $true)]
    [string]$Description,
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
    try {
      if (& $Check) {
        return
      }
    } catch {
      if ($attempt -eq $Attempts) {
        throw "Timed out waiting for $Description. Last error: $($_.Exception.Message)"
      }
    }
    Start-Sleep -Seconds $DelaySeconds
  }

  throw "Timed out waiting for $Description"
}

try {
  $env:APP_ENV = "production"
  $env:BACKEND_PORT = $BackendPort
  $env:FRONTEND_PORT = $FrontendPort
  $env:JWT_SECRET = $JwtSecret
  $env:DEFAULT_ADMIN_PASSWORD = $BootstrapPassword
  $env:LOGIN_ATTEMPT_STORE = "redis"

  Invoke-Compose -Arguments @("up", "-d", "--build", "postgres", "redis", "core-agent", "backend", "frontend")

  Wait-UntilReady -Description "backend readiness" -Check {
    $ready = Invoke-JsonRequest -Method "GET" -Uri "$BackendBaseUrl/ready"
    return $ready.code -eq 0 -and $ready.data.checks.database -eq "up" -and $ready.data.checks.agent -eq "up"
  }

  Wait-UntilReady -Description "frontend startup" -Check {
    $response = Invoke-WebRequest -Uri $FrontendBaseUrl -SkipHttpErrorCheck
    return $response.StatusCode -eq 200
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
      Invoke-Compose -Arguments @("ps")
    } catch {
      Write-Warning $_.Exception.Message
    }
    try {
      Invoke-Compose -Arguments @("logs", "--no-color", "--tail", "200")
    } catch {
      Write-Warning $_.Exception.Message
    }
  }

  try {
    Invoke-Compose -Arguments @("down", "-v", "--remove-orphans")
  } catch {
    Write-Warning $_.Exception.Message
  }
}
