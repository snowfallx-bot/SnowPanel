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

  return $response.Content | ConvertFrom-Json -Depth $JsonDepth
}

function Invoke-ApiRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Method,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [object]$Body = $null,
    [hashtable]$Headers = @{}
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
  $json = $null
  if (-not [string]::IsNullOrWhiteSpace($response.Content)) {
    try {
      $json = $response.Content | ConvertFrom-Json -Depth 20
    } catch {
      $json = $null
    }
  }

  return [PSCustomObject]@{
    StatusCode = [int]$response.StatusCode
    Content    = [string]$response.Content
    Json       = $json
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
    $response = Invoke-WebRequest -Uri $FrontendBaseUrl -SkipHttpErrorCheck
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
