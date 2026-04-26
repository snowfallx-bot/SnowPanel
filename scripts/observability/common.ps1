function Invoke-ObservabilityJsonRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Method,
    [Parameter(Mandatory = $true)]
    [string]$Uri,
    [hashtable]$Headers = @{},
    [object]$Body = $null,
    [int[]]$ExpectedStatusCodes = @(200),
    [int]$JsonDepth = 20
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

  if ($null -ne $Body) {
    $requestParams.ContentType = "application/json"
    $requestParams.Body = ($Body | ConvertTo-Json -Depth 10 -Compress)
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

    $response = [PSCustomObject]@{
      StatusCode = [int]$exceptionResponse.StatusCode
      Content    = $content
    }
  }

  if ($response.StatusCode -notin $ExpectedStatusCodes) {
    throw "Expected status $($ExpectedStatusCodes -join ', ') from $Method $Uri, got $($response.StatusCode). Body: $($response.Content)"
  }

  if ([string]::IsNullOrWhiteSpace($response.Content)) {
    return $null
  }

  return $response.Content | ConvertFrom-Json -Depth $JsonDepth
}

function Wait-ObservabilityCondition {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Description,
    [Parameter(Mandatory = $true)]
    [int]$TimeoutSeconds,
    [Parameter(Mandatory = $true)]
    [scriptblock]$Check,
    [int]$IntervalSeconds = 2,
    [string]$TimeoutMessage = ""
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    if (& $Check) {
      return
    }
    Start-Sleep -Seconds $IntervalSeconds
  }

  if (-not [string]::IsNullOrWhiteSpace($TimeoutMessage)) {
    throw $TimeoutMessage
  }

  throw "Timed out waiting for $Description within ${TimeoutSeconds}s."
}
