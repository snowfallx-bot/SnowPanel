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

function Set-ProcessEnvironmentVariables {
  param(
    [Parameter(Mandatory = $true)]
    [hashtable]$Variables
  )

  foreach ($entry in $Variables.GetEnumerator()) {
    $name = [string]$entry.Key
    if ([string]::IsNullOrWhiteSpace($name)) {
      continue
    }

    $value = $entry.Value
    if ($null -eq $value) {
      Remove-Item -Path "Env:$name" -ErrorAction SilentlyContinue
      continue
    }

    [System.Environment]::SetEnvironmentVariable($name, [string]$value, "Process")
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

function Get-JsonHttpBodyOptions {
  param(
    [object]$Body = $null,
    [int]$JsonDepth = 10
  )

  if ($null -eq $Body) {
    return [PSCustomObject]@{
      RequestBody = ""
      ContentType = ""
    }
  }

  return [PSCustomObject]@{
    RequestBody = (ConvertTo-Json -InputObject $Body -Depth $JsonDepth -Compress)
    ContentType = "application/json"
  }
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

  $requestOptions = Get-JsonHttpBodyOptions -Body $Body
  $response = Invoke-WebRequestCompat -Method $Method -Uri $Uri -Headers $Headers -ContentType $requestOptions.ContentType -Body $requestOptions.RequestBody
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

  $requestOptions = Get-JsonHttpBodyOptions -Body $Body
  $response = Invoke-WebRequestCompat -Method $Method -Uri $Uri -Headers $Headers -ContentType $requestOptions.ContentType -Body $requestOptions.RequestBody

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

function Wait-ApiStatus {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Description,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [int[]]$ExpectedStatusCodes = @(200),
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  Wait-UntilReady -Description $Description -Attempts $Attempts -DelaySeconds $DelaySeconds -Check {
    $response = Invoke-ApiRequest -Method "GET" -Uri $Uri -ExpectedStatusCodes $ExpectedStatusCodes
    return $response.StatusCode -in $ExpectedStatusCodes
  }
}

function Wait-JsonReady {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Description,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [scriptblock]$Predicate = { param($json) return $null -ne $json },
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  Wait-UntilReady -Description $Description -Attempts $Attempts -DelaySeconds $DelaySeconds -Check {
    $json = Invoke-JsonRequest -Method "GET" -Uri $Uri
    return & $Predicate $json
  }
}

function Wait-ApiJsonReady {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Description,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [int[]]$ExpectedStatusCodes = @(200),
    [scriptblock]$Predicate = { param($json) return $null -ne $json },
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  Wait-UntilReady -Description $Description -Attempts $Attempts -DelaySeconds $DelaySeconds -Check {
    $response = Invoke-ApiRequest -Method "GET" -Uri $Uri -ExpectedStatusCodes $ExpectedStatusCodes
    if ($response.StatusCode -notin $ExpectedStatusCodes) {
      return $false
    }

    return & $Predicate $response.Json
  }
}

function Test-BackendReadyChecks {
  param(
    [Parameter(Mandatory = $false)]
    [object]$Envelope
  )

  return $null -ne $Envelope -and
    [int]$Envelope.code -eq 0 -and
    [string]$Envelope.data.checks.database -eq "up" -and
    [string]$Envelope.data.checks.agent -eq "up"
}

function Wait-BackendReadyJson {
  param(
    [Parameter(Mandatory = $true)]
    [string]$BackendBaseUrl,
    [string]$Description = "backend readiness",
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  Wait-JsonReady -Description $Description -Uri "$BackendBaseUrl/ready" -Attempts $Attempts -DelaySeconds $DelaySeconds -Predicate {
    param($ready)
    return Test-BackendReadyChecks -Envelope $ready
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

  Wait-ApiJsonReady -Description $Description -Uri "$BackendBaseUrl/ready" -ExpectedStatusCodes @(200) -Attempts $Attempts -DelaySeconds $DelaySeconds -Predicate {
    param($readyJson)
    return Test-BackendReadyChecks -Envelope $readyJson
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

  Wait-ApiStatus -Description $Description -Uri $FrontendBaseUrl -ExpectedStatusCodes @(200) -Attempts $Attempts -DelaySeconds $DelaySeconds
}

function Wait-FrontendProxyHealth {
  param(
    [Parameter(Mandatory = $true)]
    [string]$FrontendBaseUrl,
    [string]$Description = "frontend proxy health",
    [int]$Attempts = 60,
    [int]$DelaySeconds = 2
  )

  Wait-JsonReady -Description $Description -Uri "$FrontendBaseUrl/health" -Attempts $Attempts -DelaySeconds $DelaySeconds -Predicate {
    param($proxyHealth)
    return Test-BackendReadyChecks -Envelope $proxyHealth
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
