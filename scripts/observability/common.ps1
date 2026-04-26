$ciCommonScript = Join-Path $PSScriptRoot "..\ci\common.ps1"
if (-not (Test-Path -LiteralPath $ciCommonScript)) {
  throw "Shared CI common script not found: $ciCommonScript"
}

. $ciCommonScript

function Invoke-ObservabilityApiRequest {
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

  return Invoke-ApiRequest `
    -Method $Method `
    -Uri $Uri `
    -Headers $Headers `
    -Body $Body `
    -ExpectedStatusCodes $ExpectedStatusCodes `
    -JsonDepth $JsonDepth
}

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

  return Invoke-JsonRequest `
    -Method $Method `
    -Uri $Uri `
    -Headers $Headers `
    -Body $Body `
    -ExpectedStatusCodes $ExpectedStatusCodes `
    -JsonDepth $JsonDepth
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

  $attempts = [Math]::Max(1, [int][Math]::Ceiling($TimeoutSeconds / [double][Math]::Max(1, $IntervalSeconds)))
  try {
    Wait-UntilReady -Description $Description -Attempts $attempts -DelaySeconds $IntervalSeconds -Check $Check
  } catch {
    if (-not [string]::IsNullOrWhiteSpace($TimeoutMessage)) {
      throw $TimeoutMessage
    }
    throw
  }
}
