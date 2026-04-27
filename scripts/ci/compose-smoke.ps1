$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

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

Assert-DockerAvailable -ScriptPath "scripts/ci/compose-smoke.ps1"

try {
  Set-ProcessEnvironmentVariables -Variables @{
    APP_ENV                = "production"
    BACKEND_PORT           = $BackendPort
    FRONTEND_PORT          = $FrontendPort
    JWT_SECRET             = $JwtSecret
    DEFAULT_ADMIN_PASSWORD = $BootstrapPassword
    LOGIN_ATTEMPT_STORE    = "redis"
  }

  Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("up", "-d", "--build", "postgres", "redis", "core-agent", "backend", "frontend")

  Wait-BackendReadyJson -BackendBaseUrl $BackendBaseUrl
  Wait-FrontendStartup -FrontendBaseUrl $FrontendBaseUrl
  Wait-FrontendProxyHealth -FrontendBaseUrl $FrontendBaseUrl

  $loginResult = Invoke-BootstrapLogin -LoginBaseUrl $FrontendBaseUrl -BootstrapPassword $BootstrapPassword
  $bootstrapAccessToken = $loginResult.BootstrapAccessToken
  $bootstrapRefreshToken = $loginResult.BootstrapRefreshToken
  $bootstrapHeaders = $loginResult.BootstrapHeaders

  $meEnvelope = Invoke-JsonRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/auth/me" -Headers $bootstrapHeaders
  Assert-True ($meEnvelope.code -eq 0) "Auth me should succeed before password change"
  Assert-True ($meEnvelope.data.username -eq "admin") "Auth me returned unexpected user"

  $passwordGateResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/dashboard/summary" -Headers $bootstrapHeaders -ExpectedStatusCodes @(403)
  Assert-True ($passwordGateResponse.StatusCode -eq 403) "Dashboard should be blocked until bootstrap password is rotated"

  $rotationResult = Invoke-BootstrapPasswordRotation -ApiBaseUrl $BackendBaseUrl -BootstrapPassword $BootstrapPassword -RotatedPassword $RotatedPassword -BootstrapAccessToken $bootstrapAccessToken
  $changePasswordEnvelope = $rotationResult.ChangePasswordEnvelope
  $rotatedAccessToken = $rotationResult.RotatedAccessToken
  $rotatedRefreshToken = $rotationResult.RotatedRefreshToken
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

  $staleRefreshResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/refresh" -Body @{
    refresh_token = $rotatedRefreshToken
  } -ExpectedStatusCodes @(401)
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

  $postLogoutResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/auth/me" -Headers $authHeaders -ExpectedStatusCodes @(401)
  Assert-True ($postLogoutResponse.StatusCode -eq 401) "Access token should be rejected after logout"

  $refreshAfterLogoutResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/auth/refresh" -Body @{
    refresh_token = $activeRefreshToken
  } -ExpectedStatusCodes @(401)
  Assert-True ($refreshAfterLogoutResponse.StatusCode -eq 401) "Refresh token should be rejected after logout"

  $Completed = $true
  Write-Host "Compose smoke test passed."
} finally {
  if (-not $Completed) {
    Write-Host "Smoke test failed, printing compose status and logs..."
    Show-ComposeDiagnostics -ComposeArgs $ComposeArgs
  }

  Stop-ComposeStack -ComposeArgs $ComposeArgs
}
