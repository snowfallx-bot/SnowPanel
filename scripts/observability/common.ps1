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

function Resolve-AlertmanagerReceiver {
  param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("critical", "warning")]
    [string]$Severity
  )

  switch ($Severity) {
    "critical" { return "snowpanel-critical" }
    "warning" { return "snowpanel-warning" }
    default { throw "Unsupported severity '$Severity' for default receiver resolution." }
  }
}

function ConvertTo-AlertmanagerFilterQuery {
  param(
    [Parameter(Mandatory = $true)]
    [hashtable]$Labels
  )

  $filters = @()
  foreach ($entry in $Labels.GetEnumerator()) {
    $key = [string]$entry.Key
    $value = [string]$entry.Value
    if ([string]::IsNullOrWhiteSpace($key) -or [string]::IsNullOrWhiteSpace($value)) {
      continue
    }

    $filters += "filter=$([System.Uri]::EscapeDataString("$key=$value"))"
  }

  return [string]::Join("&", $filters)
}

function Get-AlertmanagerReceiverNames {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Alert
  )

  if ($null -eq $Alert.receivers) {
    return @()
  }

  $names = @()
  foreach ($receiver in $Alert.receivers) {
    $name = if ($receiver -is [string]) { [string]$receiver } else { [string]$receiver.name }
    if (-not [string]::IsNullOrWhiteSpace($name)) {
      $names += $name
    }
  }

  return $names
}

function Test-AlertmanagerHasReceiver {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Alert,
    [Parameter(Mandatory = $true)]
    [string]$ReceiverName
  )

  foreach ($name in (Get-AlertmanagerReceiverNames -Alert $Alert)) {
    if ($name -ieq $ReceiverName) {
      return $true
    }
  }

  return $false
}

function New-AlertmanagerSyntheticAlert {
  param(
    [Parameter(Mandatory = $true)]
    [string]$AlertName,
    [Parameter(Mandatory = $true)]
    [ValidateSet("critical", "warning")]
    [string]$Severity,
    [Parameter(Mandatory = $true)]
    [string]$Instance,
    [Parameter(Mandatory = $true)]
    [string]$Source,
    [Parameter(Mandatory = $true)]
    [string]$Summary,
    [Parameter(Mandatory = $true)]
    [string]$Description,
    [Parameter(Mandatory = $true)]
    [DateTimeOffset]$StartsAt,
    [Parameter(Mandatory = $true)]
    [DateTimeOffset]$EndsAt
  )

  return @{
    labels = @{
      alertname = $AlertName
      severity  = $Severity
      instance  = $Instance
      source    = $Source
    }
    annotations = @{
      summary     = $Summary
      description = $Description
    }
    startsAt = $StartsAt.ToString("o")
    endsAt   = $EndsAt.ToString("o")
  }
}

function Get-AlertmanagerApiUriWithFilters {
  param(
    [Parameter(Mandatory = $true)]
    [string]$AlertmanagerBaseUrl,
    [Parameter(Mandatory = $true)]
    [string]$ApiPath,
    [hashtable]$Labels = @{}
  )

  $filterQuery = ConvertTo-AlertmanagerFilterQuery -Labels $Labels
  if ([string]::IsNullOrWhiteSpace($filterQuery)) {
    return "$AlertmanagerBaseUrl$ApiPath?active=true"
  }

  return "$AlertmanagerBaseUrl$ApiPath?active=true&$filterQuery"
}

function Get-AlertmanagerActiveAlerts {
  param(
    [Parameter(Mandatory = $true)]
    [string]$AlertmanagerBaseUrl,
    [hashtable]$Labels = @{}
  )

  $uri = Get-AlertmanagerApiUriWithFilters -AlertmanagerBaseUrl $AlertmanagerBaseUrl -ApiPath "/api/v2/alerts" -Labels $Labels
  return Invoke-ObservabilityJsonRequest -Method "GET" -Uri $uri -ExpectedStatusCodes @(200)
}

function Get-AlertmanagerActiveAlertGroups {
  param(
    [Parameter(Mandatory = $true)]
    [string]$AlertmanagerBaseUrl,
    [hashtable]$Labels = @{}
  )

  $uri = Get-AlertmanagerApiUriWithFilters -AlertmanagerBaseUrl $AlertmanagerBaseUrl -ApiPath "/api/v2/alerts/groups" -Labels $Labels
  return Invoke-ObservabilityJsonRequest -Method "GET" -Uri $uri -ExpectedStatusCodes @(200)
}

function Submit-AlertmanagerAlerts {
  param(
    [Parameter(Mandatory = $true)]
    [string]$AlertmanagerBaseUrl,
    [Parameter(Mandatory = $true)]
    [object[]]$Alerts
  )

  return Invoke-ObservabilityJsonRequest -Method "POST" -Uri "$AlertmanagerBaseUrl/api/v2/alerts" -Body $Alerts -ExpectedStatusCodes @(200, 202)
}
