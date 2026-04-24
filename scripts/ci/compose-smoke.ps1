$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$ProjectName = "snowpanel-smoke"
$BackendPort = "18080"
$FrontendPort = "15173"
$BackendBaseUrl = "http://127.0.0.1:$BackendPort"
$FrontendBaseUrl = "http://127.0.0.1:$FrontendPort"
$BootstrapPassword = "BootstrapSmoke1!"
$RotatedPassword = "BootstrapSmoke2!"
$JwtSecret = "SmokeSecret_2026_Integration_Check_123!"
$SmokeFilePath = "/tmp/snowpanel-compose-smoke.txt"
$RenamedSmokeFilePath = "/tmp/snowpanel-compose-smoke-renamed.txt"
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

function Assert-True {
  param(
    [Parameter(Mandatory = $true)]
    [bool]$Condition,
    [Parameter(Mandatory = $true)]
    [string]$Message
  )

  if (-not $Condition) {
    throw $Message
  }
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

  $loginEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$FrontendBaseUrl/api/v1/auth/login" -Body @{
    username = "admin"
    password = $BootstrapPassword
  }
  Assert-True ($loginEnvelope.code -eq 0) "Login via frontend proxy failed"
  Assert-True ($loginEnvelope.data.user.must_change_password -eq $true) "Bootstrap admin should require password rotation on first login"

  $bootstrapAccessToken = [string]$loginEnvelope.data.access_token
  $bootstrapRefreshToken = [string]$loginEnvelope.data.refresh_token
  $bootstrapHeaders = @{ Authorization = "Bearer $bootstrapAccessToken" }

  $meEnvelope = Invoke-JsonRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/auth/me" -Headers $bootstrapHeaders
  Assert-True ($meEnvelope.code -eq 0) "Auth me should succeed before password change"
  Assert-True ($meEnvelope.data.username -eq "admin") "Auth me returned unexpected user"

  $passwordGateResponse = Invoke-WebRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/dashboard/summary" -Headers $bootstrapHeaders -SkipHttpErrorCheck
  Assert-True ($passwordGateResponse.StatusCode -eq 403) "Dashboard should be blocked until bootstrap password is rotated"

  $changePasswordEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/change-password" -Headers $bootstrapHeaders -Body @{
    current_password = $BootstrapPassword
    new_password = $RotatedPassword
  }
  Assert-True ($changePasswordEnvelope.code -eq 0) "Password change failed"
  Assert-True ($changePasswordEnvelope.data.user.must_change_password -eq $false) "Rotated session should clear must_change_password"

  $rotatedAccessToken = [string]$changePasswordEnvelope.data.access_token
  $rotatedRefreshToken = [string]$changePasswordEnvelope.data.refresh_token
  Assert-True ($rotatedAccessToken -ne $bootstrapAccessToken) "Password change should rotate access token"
  Assert-True ($rotatedRefreshToken -ne $bootstrapRefreshToken) "Password change should rotate refresh token"

  $refreshEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/refresh" -Body @{
    refresh_token = $rotatedRefreshToken
  }
  Assert-True ($refreshEnvelope.code -eq 0) "Refresh token exchange failed"
  Assert-True ($refreshEnvelope.data.access_token -ne $rotatedAccessToken) "Refresh should rotate access token"
  Assert-True ($refreshEnvelope.data.refresh_token -ne $rotatedRefreshToken) "Refresh should rotate refresh token"

  $activeAccessToken = [string]$refreshEnvelope.data.access_token
  $activeRefreshToken = [string]$refreshEnvelope.data.refresh_token
  $authHeaders = @{ Authorization = "Bearer $activeAccessToken" }

  $staleRefreshResponse = Invoke-WebRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/refresh" -ContentType "application/json" -Body (@{ refresh_token = $rotatedRefreshToken } | ConvertTo-Json -Compress) -SkipHttpErrorCheck
  Assert-True ($staleRefreshResponse.StatusCode -eq 401) "Old refresh token should be rejected after rotation"

  $dashboardEnvelope = Invoke-JsonRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/dashboard/summary" -Headers $authHeaders
  Assert-True ($dashboardEnvelope.code -eq 0) "Dashboard summary should succeed after password rotation"
  Assert-True (-not [string]::IsNullOrWhiteSpace([string]$dashboardEnvelope.data.hostname)) "Dashboard summary returned empty hostname"

  $listEnvelope = Invoke-JsonRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/files/list?path=%2Ftmp" -Headers $authHeaders
  Assert-True ($listEnvelope.code -eq 0) "File listing for /tmp should succeed"

  $writeEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/files/write" -Headers $authHeaders -Body @{
    path = $SmokeFilePath
    content = "compose smoke content"
    create_if_not_exists = $true
    truncate = $true
    encoding = "utf-8"
  }
  Assert-True ($writeEnvelope.code -eq 0) "Writing smoke file failed"

  $readEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/files/read" -Headers $authHeaders -Body @{
    path = $SmokeFilePath
    max_bytes = 1024
    encoding = "utf-8"
  }
  Assert-True ($readEnvelope.code -eq 0) "Reading smoke file failed"
  Assert-True ($readEnvelope.data.content -eq "compose smoke content") "Smoke file content mismatch"

  $renameEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/files/rename" -Headers $authHeaders -Body @{
    source_path = $SmokeFilePath
    target_path = $RenamedSmokeFilePath
  }
  Assert-True ($renameEnvelope.code -eq 0) "Renaming smoke file failed"

  $deleteEnvelope = Invoke-JsonRequest -Method "DELETE" -Uri "$BackendBaseUrl/api/v1/files/delete" -Headers $authHeaders -Body @{
    path = $RenamedSmokeFilePath
    recursive = $false
  }
  Assert-True ($deleteEnvelope.code -eq 0) "Deleting smoke file failed"

  $logoutEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/logout" -Headers $authHeaders
  Assert-True ($logoutEnvelope.code -eq 0) "Logout failed"

  $postLogoutResponse = Invoke-WebRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/auth/me" -Headers $authHeaders -SkipHttpErrorCheck
  Assert-True ($postLogoutResponse.StatusCode -eq 401) "Access token should be rejected after logout"

  $refreshAfterLogoutResponse = Invoke-WebRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/refresh" -ContentType "application/json" -Body (@{ refresh_token = $activeRefreshToken } | ConvertTo-Json -Compress) -SkipHttpErrorCheck
  Assert-True ($refreshAfterLogoutResponse.StatusCode -eq 401) "Refresh token should be rejected after logout"

  $Completed = $true
  Write-Host "Compose smoke test passed."
} finally {
  if (-not $Completed) {
    Write-Host "Smoke test failed, printing compose status and logs..."
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
