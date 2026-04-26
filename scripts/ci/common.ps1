function Invoke-ComposeCommand {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$ComposeArgs,
    [Parameter(Mandatory = $true)]
    [string[]]$Arguments
  )

  & docker @ComposeArgs @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "docker compose $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
  }
}

function Show-ComposeDiagnostics {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$ComposeArgs,
    [int]$TailLines = 200
  )

  try {
    Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("ps")
  } catch {
    Write-Warning $_.Exception.Message
  }

  try {
    Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("logs", "--no-color", "--tail", "$TailLines")
  } catch {
    Write-Warning $_.Exception.Message
  }
}

function Stop-ComposeStack {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$ComposeArgs
  )

  try {
    Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("down", "-v", "--remove-orphans")
  } catch {
    Write-Warning $_.Exception.Message
  }
}

function Assert-DockerAvailable {
  param(
    [Parameter(Mandatory = $true)]
    [string]$ScriptPath
  )

  if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "docker command not found. Install Docker/Compose before running $ScriptPath."
  }
}

function Convert-HeaderValueToString {
  param(
    [Parameter(Mandatory = $false)]
    [object]$Value
  )

  if ($null -eq $Value) {
    return ""
  }

  if ($Value -is [System.Array]) {
    return (($Value | ForEach-Object { [string]$_ }) -join ", ")
  }

  return [string]$Value
}

function Invoke-WebRequestCompat {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Method,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [hashtable]$Headers = @{},
    [string]$ContentType = "",
    [string]$Body = ""
  )

  $supportsSkipHttpErrorCheck = (Get-Command Invoke-WebRequest).Parameters.ContainsKey("SkipHttpErrorCheck")
  $requestParams = @{
    Method  = $Method
    Uri     = $Uri
    Headers = $Headers
  }

  if ($supportsSkipHttpErrorCheck) {
    $requestParams.SkipHttpErrorCheck = $true
  }

  if (-not [string]::IsNullOrWhiteSpace($ContentType)) {
    $requestParams.ContentType = $ContentType
  }

  if (-not [string]::IsNullOrWhiteSpace($Body)) {
    $requestParams.Body = $Body
  }

  try {
    $response = Invoke-WebRequest @requestParams
  } catch {
    if ($supportsSkipHttpErrorCheck) {
      throw
    }

    $exceptionResponse = $_.Exception.Response
    if ($null -eq $exceptionResponse) {
      throw
    }

    $stream = $exceptionResponse.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($stream)
    $content = $reader.ReadToEnd()
    $reader.Dispose()
    $stream.Dispose()

    $responseHeaders = @{}
    foreach ($headerName in $exceptionResponse.Headers.AllKeys) {
      $responseHeaders[[string]$headerName] = Convert-HeaderValueToString -Value $exceptionResponse.Headers.GetValues($headerName)
    }

    $response = [PSCustomObject]@{
      StatusCode = [int]$exceptionResponse.StatusCode
      Content    = [string]$content
      Headers    = $responseHeaders
    }
  }

  $headersMap = @{}
  if ($null -ne $response.Headers) {
    foreach ($headerName in $response.Headers.Keys) {
      $headersMap[[string]$headerName] = Convert-HeaderValueToString -Value $response.Headers[$headerName]
    }
  }

  return [PSCustomObject]@{
    StatusCode = [int]$response.StatusCode
    Content    = [string]$response.Content
    Headers    = $headersMap
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
    [int[]]$ExpectedStatusCodes = @(200),
    [int]$JsonDepth = 20
  )

  $requestBody = ""
  $contentType = ""
  if ($null -ne $Body) {
    $requestBody = (ConvertTo-Json -InputObject $Body -Depth 10 -Compress)
    $contentType = "application/json"
  }

  $response = Invoke-WebRequestCompat -Method $Method -Uri $Uri -Headers $Headers -ContentType $contentType -Body $requestBody
  if ($response.StatusCode -notin $ExpectedStatusCodes) {
    throw "Expected status $($ExpectedStatusCodes -join ', ') from $Method $Uri, got $($response.StatusCode). Body: $($response.Content)"
  }

  if ([string]::IsNullOrWhiteSpace($response.Content)) {
    return $null
  }

  return $response.Content | ConvertFrom-Json -Depth $JsonDepth
}

function Invoke-ApiRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Method,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [object]$Body = $null,
    [hashtable]$Headers = @{},
    [int[]]$ExpectedStatusCodes = @(),
    [int]$JsonDepth = 20
  )

  $requestBody = ""
  $contentType = ""
  if ($null -ne $Body) {
    $requestBody = (ConvertTo-Json -InputObject $Body -Depth 10 -Compress)
    $contentType = "application/json"
  }

  $response = Invoke-WebRequestCompat -Method $Method -Uri $Uri -Headers $Headers -ContentType $contentType -Body $requestBody

  if ($ExpectedStatusCodes.Count -gt 0 -and $response.StatusCode -notin $ExpectedStatusCodes) {
    throw "Expected status $($ExpectedStatusCodes -join ', ') from $Method $Uri, got $($response.StatusCode). Body: $($response.Content)"
  }

  $json = $null
  if (-not [string]::IsNullOrWhiteSpace($response.Content)) {
    try {
      $json = $response.Content | ConvertFrom-Json -Depth $JsonDepth
    } catch {
      $json = $null
    }
  }

  return [PSCustomObject]@{
    StatusCode = [int]$response.StatusCode
    Content    = [string]$response.Content
    Json       = $json
    Headers    = $response.Headers
  }
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

function Wait-BackendReadyJson {
  param(
    [Parameter(Mandatory = $true)]
    [string]$BackendBaseUrl,
    [string]$Description = "backend readiness",
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  Wait-UntilReady -Description $Description -Attempts $Attempts -DelaySeconds $DelaySeconds -Check {
    $ready = Invoke-JsonRequest -Method "GET" -Uri "$BackendBaseUrl/ready"
    return $ready.code -eq 0 -and $ready.data.checks.database -eq "up" -and $ready.data.checks.agent -eq "up"
  }
}

function Wait-BackendReadyApi {
  param(
    [Parameter(Mandatory = $true)]
    [string]$BackendBaseUrl,
    [string]$Description = "backend readiness",
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  Wait-UntilReady -Description $Description -Attempts $Attempts -DelaySeconds $DelaySeconds -Check {
    $ready = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/ready"
    return $ready.StatusCode -eq 200 -and
      $null -ne $ready.Json -and
      [int]$ready.Json.code -eq 0 -and
      [string]$ready.Json.data.checks.database -eq "up" -and
      [string]$ready.Json.data.checks.agent -eq "up"
  }
}

function Wait-FrontendStartup {
  param(
    [Parameter(Mandatory = $true)]
    [string]$FrontendBaseUrl,
    [string]$Description = "frontend startup",
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  Wait-UntilReady -Description $Description -Attempts $Attempts -DelaySeconds $DelaySeconds -Check {
    $response = Invoke-WebRequestCompat -Method "GET" -Uri $FrontendBaseUrl
    return $response.StatusCode -eq 200
  }
}

function Wait-FrontendProxyHealth {
  param(
    [Parameter(Mandatory = $true)]
    [string]$FrontendBaseUrl,
    [string]$Description = "frontend proxy health",
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  Wait-UntilReady -Description $Description -Attempts $Attempts -DelaySeconds $DelaySeconds -Check {
    $proxyHealth = Invoke-JsonRequest -Method "GET" -Uri "$FrontendBaseUrl/health"
    return $proxyHealth.code -eq 0 -and $proxyHealth.data.checks.database -eq "up" -and $proxyHealth.data.checks.agent -eq "up"
  }
}

function Invoke-BootstrapLogin {
  param(
    [Parameter(Mandatory = $true)]
    [string]$LoginBaseUrl,
    [Parameter(Mandatory = $true)]
    [string]$BootstrapPassword,
    [string]$Username = "admin"
  )

  $loginEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$LoginBaseUrl/api/v1/auth/login" -Body @{
    username = $Username
    password = $BootstrapPassword
  }
  Assert-True ($loginEnvelope.code -eq 0) "Bootstrap login failed for user '$Username'"
  Assert-True ([bool]$loginEnvelope.data.user.must_change_password) "Bootstrap user '$Username' should require password rotation"

  $bootstrapAccessToken = [string]$loginEnvelope.data.access_token
  $bootstrapRefreshToken = [string]$loginEnvelope.data.refresh_token
  Assert-True (-not [string]::IsNullOrWhiteSpace($bootstrapAccessToken)) "Bootstrap login returned empty access token"
  Assert-True (-not [string]::IsNullOrWhiteSpace($bootstrapRefreshToken)) "Bootstrap login returned empty refresh token"

  return [PSCustomObject]@{
    LoginEnvelope         = $loginEnvelope
    BootstrapAccessToken  = $bootstrapAccessToken
    BootstrapRefreshToken = $bootstrapRefreshToken
    BootstrapHeaders      = @{ Authorization = "Bearer $bootstrapAccessToken" }
  }
}

function Invoke-BootstrapPasswordRotation {
  param(
    [Parameter(Mandatory = $true)]
    [string]$ApiBaseUrl,
    [Parameter(Mandatory = $true)]
    [string]$BootstrapPassword,
    [Parameter(Mandatory = $true)]
    [string]$RotatedPassword,
    [Parameter(Mandatory = $true)]
    [string]$BootstrapAccessToken,
    [string]$Username = "admin"
  )

  $bootstrapHeaders = @{ Authorization = "Bearer $BootstrapAccessToken" }
  $changePasswordEnvelope = Invoke-JsonRequest -Method "POST" -Uri "$ApiBaseUrl/api/v1/auth/change-password" -Headers $bootstrapHeaders -Body @{
    current_password = $BootstrapPassword
    new_password     = $RotatedPassword
  }
  Assert-True ($changePasswordEnvelope.code -eq 0) "Password rotation failed for user '$Username'"
  Assert-True (-not [bool]$changePasswordEnvelope.data.user.must_change_password) "Rotated user '$Username' should clear must_change_password"

  $rotatedAccessToken = [string]$changePasswordEnvelope.data.access_token
  $rotatedRefreshToken = [string]$changePasswordEnvelope.data.refresh_token
  Assert-True (-not [string]::IsNullOrWhiteSpace($rotatedAccessToken)) "Password rotation returned empty access token"
  Assert-True (-not [string]::IsNullOrWhiteSpace($rotatedRefreshToken)) "Password rotation returned empty refresh token"

  return [PSCustomObject]@{
    ChangePasswordEnvelope = $changePasswordEnvelope
    RotatedAccessToken    = $rotatedAccessToken
    RotatedRefreshToken   = $rotatedRefreshToken
    RotatedHeaders        = @{ Authorization = "Bearer $rotatedAccessToken" }
  }
}

function Initialize-BootstrapAdminSession {
  param(
    [Parameter(Mandatory = $true)]
    [string]$ApiBaseUrl,
    [Parameter(Mandatory = $true)]
    [string]$BootstrapPassword,
    [Parameter(Mandatory = $true)]
    [string]$RotatedPassword,
    [string]$LoginBaseUrl = "",
    [string]$Username = "admin"
  )

  $resolvedLoginBaseUrl = $LoginBaseUrl
  if ([string]::IsNullOrWhiteSpace($resolvedLoginBaseUrl)) {
    $resolvedLoginBaseUrl = $ApiBaseUrl
  }

  $loginResult = Invoke-BootstrapLogin -LoginBaseUrl $resolvedLoginBaseUrl -BootstrapPassword $BootstrapPassword -Username $Username
  $rotationResult = Invoke-BootstrapPasswordRotation -ApiBaseUrl $ApiBaseUrl -BootstrapPassword $BootstrapPassword -RotatedPassword $RotatedPassword -BootstrapAccessToken $loginResult.BootstrapAccessToken -Username $Username

  return [PSCustomObject]@{
    LoginEnvelope          = $loginResult.LoginEnvelope
    ChangePasswordEnvelope = $rotationResult.ChangePasswordEnvelope
    BootstrapAccessToken   = $loginResult.BootstrapAccessToken
    BootstrapRefreshToken  = $loginResult.BootstrapRefreshToken
    RotatedAccessToken     = $rotationResult.RotatedAccessToken
    RotatedRefreshToken    = $rotationResult.RotatedRefreshToken
    BootstrapHeaders       = $loginResult.BootstrapHeaders
    RotatedHeaders         = $rotationResult.RotatedHeaders
  }
}
