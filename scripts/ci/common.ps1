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
