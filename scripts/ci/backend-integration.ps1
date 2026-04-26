$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot "common.ps1")

$ProjectName = "snowpanel-backend-it"
$BackendPort = "18082"
$BackendBaseUrl = "http://127.0.0.1:$BackendPort"
$BootstrapPassword = "BackendITBootstrap1!"
$RotatedPassword = "BackendITRotated2!"
$JwtSecret = "BackendIntegrationSecret_2026_Check_123!"
$ComposeArgs = @("compose", "--project-name", $ProjectName)
$Completed = $false

Assert-DockerAvailable -ScriptPath "scripts/ci/backend-integration.ps1"

function Assert-ApiSuccess {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Response,
    [Parameter(Mandatory = $true)]
    [string]$Description
  )

  Assert-True ($Response.StatusCode -eq 200) "$($Description): expected HTTP 200, got $($Response.StatusCode). Body: $($Response.Content)"
  Assert-True ($null -ne $Response.Json) "$($Description): expected JSON response body"
  Assert-True ([int]$Response.Json.code -eq 0) "$($Description): expected envelope code 0, got $($Response.Json.code). Body: $($Response.Content)"
}

function Assert-SuccessOrKnownFailure {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Response,
    [Parameter(Mandatory = $true)]
    [string]$Description,
    [int[]]$AllowedFailureStatusCodes = @(502, 503),
    [int[]]$AllowedFailureCodes = @(),
    [scriptblock]$OnSuccess = { param($Json) return $true }
  )

  if ($Response.StatusCode -eq 200) {
    Assert-ApiSuccess -Response $Response -Description $Description
    $ok = & $OnSuccess $Response.Json
    Assert-True ([bool]$ok) "$($Description): success payload validation failed. Body: $($Response.Content)"
    return
  }

  Assert-True ($Response.StatusCode -in $AllowedFailureStatusCodes) "$($Description): unexpected HTTP status $($Response.StatusCode). Body: $($Response.Content)"
  Assert-True ($null -ne $Response.Json) "$($Description): expected JSON error payload"
  $errorCode = [int]$Response.Json.code
  Assert-True ($errorCode -gt 0) "$($Description): expected non-zero error code. Body: $($Response.Content)"
  if ($AllowedFailureCodes.Count -gt 0) {
    Assert-True ($errorCode -in $AllowedFailureCodes) "$($Description): unexpected error code $errorCode. Body: $($Response.Content)"
  }
}

function Wait-ForTaskTerminal {
  param(
    [Parameter(Mandatory = $true)]
    [string]$BaseUrl,
    [Parameter(Mandatory = $true)]
    [hashtable]$Headers,
    [Parameter(Mandatory = $true)]
    [long]$TaskID,
    [int]$Attempts = 40,
    [int]$DelaySeconds = 1
  )

  for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
    $detailResponse = Invoke-ApiRequest -Method "GET" -Uri "$BaseUrl/api/v1/tasks/$TaskID" -Headers $Headers
    Assert-ApiSuccess -Response $detailResponse -Description "fetch task detail $TaskID"
    $status = [string]$detailResponse.Json.data.summary.status
    if ($status -in @("success", "failed", "canceled")) {
      return $detailResponse.Json.data
    }
    Start-Sleep -Seconds $DelaySeconds
  }

  throw "Timed out waiting for task $TaskID to reach terminal status"
}

try {
  $env:APP_ENV = "production"
  $env:BACKEND_PORT = $BackendPort
  $env:JWT_SECRET = $JwtSecret
  $env:DEFAULT_ADMIN_PASSWORD = $BootstrapPassword
  $env:LOGIN_ATTEMPT_STORE = "redis"

  Invoke-ComposeCommand -ComposeArgs $ComposeArgs -Arguments @("up", "-d", "--build", "postgres", "redis", "core-agent", "backend")

  Wait-BackendReadyApi -BackendBaseUrl $BackendBaseUrl

  $bootstrapSession = Initialize-BootstrapAdminSession -ApiBaseUrl $BackendBaseUrl -BootstrapPassword $BootstrapPassword -RotatedPassword $RotatedPassword
  $authHeaders = $bootstrapSession.RotatedHeaders

  $servicesResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/services" -Headers $authHeaders
  Assert-SuccessOrKnownFailure -Response $servicesResponse -Description "list services" -AllowedFailureStatusCodes @(502, 503) -AllowedFailureCodes @(3001, 5001, 5003) -OnSuccess {
    param($json)
    return $null -ne $json.data.services
  }

  $disallowedServiceResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/services/not-allowed/restart" -Headers $authHeaders
  Assert-True ($disallowedServiceResponse.StatusCode -eq 403) "restart disallowed service should be forbidden"
  Assert-True ($null -ne $disallowedServiceResponse.Json -and [int]$disallowedServiceResponse.Json.code -eq 5002) "restart disallowed service should return agent code 5002"

  $dockerContainersResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/docker/containers" -Headers $authHeaders
  Assert-SuccessOrKnownFailure -Response $dockerContainersResponse -Description "list docker containers" -AllowedFailureStatusCodes @(502, 503) -AllowedFailureCodes @(3001, 6000, 6003) -OnSuccess {
    param($json)
    return $null -ne $json.data.containers
  }

  $dockerImagesResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/docker/images" -Headers $authHeaders
  Assert-SuccessOrKnownFailure -Response $dockerImagesResponse -Description "list docker images" -AllowedFailureStatusCodes @(502, 503) -AllowedFailureCodes @(3001, 6000, 6003) -OnSuccess {
    param($json)
    return $null -ne $json.data.images
  }

  $invalidContainerResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/docker/containers/bad%2Aid/start" -Headers $authHeaders
  Assert-True ($invalidContainerResponse.StatusCode -eq 400) "invalid docker container id should return HTTP 400"
  Assert-True ($null -ne $invalidContainerResponse.Json -and [int]$invalidContainerResponse.Json.code -eq 6001) "invalid docker container id should return agent code 6001"

  $cronListResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/cron" -Headers $authHeaders
  Assert-SuccessOrKnownFailure -Response $cronListResponse -Description "list cron tasks" -AllowedFailureStatusCodes @(502, 503) -AllowedFailureCodes @(3001, 7002) -OnSuccess {
    param($json)
    return $null -ne $json.data.tasks
  }

  $blockedCronResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/cron" -Headers $authHeaders -Body @{
    expression = "*/15 * * * *"
    command    = "backup;id"
    enabled    = $true
  }
  Assert-True ($blockedCronResponse.StatusCode -eq 400) "blocked cron command should return HTTP 400"
  Assert-True ($null -ne $blockedCronResponse.Json -and [int]$blockedCronResponse.Json.code -eq 7000) "blocked cron command should return agent code 7000"

  $createdCronID = ""
  $createCronResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/cron" -Headers $authHeaders -Body @{
    expression = "*/20 * * * *"
    command    = "backup"
    enabled    = $true
  }
  if ($createCronResponse.StatusCode -eq 200) {
    Assert-ApiSuccess -Response $createCronResponse -Description "create cron task"
    $createdCronID = [string]$createCronResponse.Json.data.task.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($createdCronID)) "created cron task id should not be empty"

    $updateCronResponse = Invoke-ApiRequest -Method "PUT" -Uri "$BackendBaseUrl/api/v1/cron/$createdCronID" -Headers $authHeaders -Body @{
      expression = "*/30 * * * *"
      command    = "cleanup"
      enabled    = $true
    }
    Assert-ApiSuccess -Response $updateCronResponse -Description "update cron task"

    $disableCronResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/cron/$createdCronID/disable" -Headers $authHeaders
    Assert-ApiSuccess -Response $disableCronResponse -Description "disable cron task"
    Assert-True (-not [bool]$disableCronResponse.Json.data.task.enabled) "cron task should be disabled"

    $enableCronResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/cron/$createdCronID/enable" -Headers $authHeaders
    Assert-ApiSuccess -Response $enableCronResponse -Description "enable cron task"
    Assert-True ([bool]$enableCronResponse.Json.data.task.enabled) "cron task should be enabled"

    $deleteCronResponse = Invoke-ApiRequest -Method "DELETE" -Uri "$BackendBaseUrl/api/v1/cron/$createdCronID" -Headers $authHeaders
    Assert-ApiSuccess -Response $deleteCronResponse -Description "delete cron task"
  } else {
    Assert-True ($createCronResponse.StatusCode -in @(502, 503)) "create cron task expected HTTP 200, 502 or 503. Got $($createCronResponse.StatusCode). Body: $($createCronResponse.Content)"
    Assert-True ($null -ne $createCronResponse.Json -and [int]$createCronResponse.Json.code -in @(3001, 7002)) "create cron task returned unexpected failure code. Body: $($createCronResponse.Content)"
  }

  $createDockerTaskResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/tasks/docker/restart" -Headers $authHeaders -Body @{
    container_id = "bad*id"
  }
  Assert-ApiSuccess -Response $createDockerTaskResponse -Description "create docker restart task"
  $taskID = [long]$createDockerTaskResponse.Json.data.id
  Assert-True ($taskID -gt 0) "created docker restart task id should be positive"

  $taskDetail = Wait-ForTaskTerminal -BaseUrl $BackendBaseUrl -Headers $authHeaders -TaskID $taskID
  Assert-True ([string]$taskDetail.summary.status -eq "failed") "docker restart task should fail for invalid container id"
  Assert-True (-not [string]::IsNullOrWhiteSpace([string]$taskDetail.summary.error_message)) "failed docker restart task should contain error message"

  $retryTaskResponse = Invoke-ApiRequest -Method "POST" -Uri "$BackendBaseUrl/api/v1/tasks/$taskID/retry" -Headers $authHeaders
  Assert-ApiSuccess -Response $retryTaskResponse -Description "retry failed task"
  $retryTaskID = [long]$retryTaskResponse.Json.data.id
  Assert-True ($retryTaskID -gt 0 -and $retryTaskID -ne $taskID) "retry should create a different task id"

  $retryTaskDetail = Wait-ForTaskTerminal -BaseUrl $BackendBaseUrl -Headers $authHeaders -TaskID $retryTaskID
  Assert-True ([string]$retryTaskDetail.summary.status -eq "failed") "retried docker restart task should fail for invalid container id"

  $tasksListResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/tasks?page=1&size=20&type=docker_restart" -Headers $authHeaders
  Assert-ApiSuccess -Response $tasksListResponse -Description "list docker restart tasks"
  Assert-True ([int64]$tasksListResponse.Json.data.total -ge 2) "task list should contain at least two docker restart tasks"

  $auditTasksResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/audit/logs?module=tasks&page=1&size=20" -Headers $authHeaders
  Assert-ApiSuccess -Response $auditTasksResponse -Description "list task audit logs"
  Assert-True ([int64]$auditTasksResponse.Json.data.total -ge 1) "task audit logs should not be empty"

  $auditDockerResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/audit/logs?module=docker&page=1&size=20" -Headers $authHeaders
  Assert-ApiSuccess -Response $auditDockerResponse -Description "list docker audit logs"
  Assert-True ([int64]$auditDockerResponse.Json.data.total -ge 1) "docker audit logs should not be empty"

  $auditServicesResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/audit/logs?module=services&page=1&size=20" -Headers $authHeaders
  Assert-ApiSuccess -Response $auditServicesResponse -Description "list services audit logs"
  Assert-True ([int64]$auditServicesResponse.Json.data.total -ge 1) "services audit logs should not be empty"

  $auditCronResponse = Invoke-ApiRequest -Method "GET" -Uri "$BackendBaseUrl/api/v1/audit/logs?module=cron&page=1&size=20" -Headers $authHeaders
  Assert-ApiSuccess -Response $auditCronResponse -Description "list cron audit logs"
  Assert-True ([int64]$auditCronResponse.Json.data.total -ge 1) "cron audit logs should not be empty"

  $Completed = $true
  Write-Host "Backend integration test passed."
} finally {
  if (-not $Completed) {
    Write-Host "Backend integration test failed, printing compose status and logs..."
    Show-ComposeDiagnostics -ComposeArgs $ComposeArgs
  }

  Stop-ComposeStack -ComposeArgs $ComposeArgs
}
