# CI Scripts

PowerShell scripts used by GitHub Actions jobs and local CI-style verification runs.

## `compose-smoke.ps1`

Brings up core services (`postgres`, `redis`, `core-agent`, `backend`, `frontend`) and validates baseline auth/session/files smoke flows.

## `backend-integration.ps1`

Runs backend integration checks against real `backend + core-agent + postgres/redis` wiring, including services/docker/cron/tasks/audit paths.

## `frontend-e2e.ps1`

Runs Playwright E2E coverage against the compose stack.

## `observability-smoke.ps1`

Brings up compose + observability stack (`prometheus`, `alertmanager`, `otel-collector`, `jaeger`) and runs end-to-end observability smoke via `scripts/observability/full-smoke.ps1`.
