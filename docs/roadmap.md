# Roadmap

Language: **English** | [简体中文](roadmap.zh-CN.md)

This roadmap reflects the current repository state rather than the original bootstrap plan.

## Completed Foundations

- Real backend <-> core-agent gRPC path across dashboard, files, services, Docker, and cron
- Host-agent deployment mode with systemd templates and Ubuntu one-click installer
- Security baseline for secrets, bootstrap admin, internal port exposure, and cron allowlists
- RBAC-backed auth/session model with permission-aware frontend gating
- Real async task execution for Docker and service restarts
- File operations expanded to practical ops workflows
- Layered CI coverage across backend tests, compose smoke, backend integration, and frontend e2e

## Completed in P2

### P2-2 Production Observability

- Prometheus metrics and baseline alert rules are in place
- Alertmanager baseline routing is in place
- OTEL tracing baseline now exists for:
  - backend HTTP spans
  - backend gRPC client spans
  - core-agent gRPC server spans
  - OTel Collector -> Jaeger pipeline
- Prometheus/Alertmanager SLO baseline has been expanded with:
  - backend availability recording rules and warning/critical SLO alerts
  - core-agent gRPC error-ratio recording rule and warning/critical alerts
  - warning/critical receiver split in Alertmanager routing baseline
- Observability smoke scripts now exist for:
  - Jaeger cross-service trace validation (`scripts/observability/trace-smoke.ps1`)
  - Alertmanager synthetic alert injection validation (`scripts/observability/alertmanager-smoke.ps1`)
  - One-shot trace + alertmanager validation (`scripts/observability/full-smoke.ps1`)
- Observability config validation gate now exists:
  - `scripts/observability/validate-config.ps1` (`promtool`/`amtool` checks with Docker-first local fallback + `promtool test rules`)
  - CI `observability-config` job in `.github/workflows/ci.yml`
- Alertmanager production rollout helpers now exist:
  - production receiver template: `deploy/observability/alertmanager/alertmanager.production.example.yml`
  - generated config workflow: `scripts/observability/generate-alertmanager-config.ps1`
- SLO burn-rate coverage now includes 5m/30m windows and warning/critical alert pairs
- Compose + host-agent observability smoke evidence is recorded in:
  - `docs/observability-validation.md`
  - `docs/observability-validation.zh-CN.md`

### P2-3 Prototype-Trace Cleanup

- Outdated backend README legacy prototype notes have been removed
- App shell and frontend e2e marker no longer use `Linux Panel Prototype`
- Root README now exposes observability commands and docs
- Legacy prototype wording and duplicate observability instructions have been aligned across README/roadmap/observability docs

## P3 Production Hardening

### P3-0 Stabilization Gate

Local P3-0 stabilization is complete on branch `p3-production-hardening`:

- `make lint` passes
- `make test` passes
- Backend, core-agent, and frontend module-level gates pass
- `make proto-go` is runnable on Windows local environments through Makefile tool auto-discovery
- Checked-in Go protobuf bindings are current with the available local proto toolchain
- Compose mode smoke passes against `/health` and `/ready`
- Host-agent mode smoke passes against `/health` and `/ready`
- Evidence and local environment notes are recorded in `docs/p3-stabilization-report.md`

CI remains the final remote confirmation for this milestone after the branch is pushed.

### P3-1 Alert Delivery & Operational Governance

Local P3-1 alerting governance is complete:

- Alerting runbook added in `docs/alerting-runbook.md`
- Warning and critical ownership, paging policy, escalation path, dedup cadence, inhibition, silence, rollback, and synthetic alert procedure are documented
- Observability docs now link to the alerting runbook
- `scripts/observability/validate-config.ps1` can validate an extra generated Alertmanager config file
- Prometheus config/rules/tests, Alertmanager baseline, production example, and generated production config validation pass
- Warning, critical, and inhibition synthetic Alertmanager smoke tests pass
- Local evidence is recorded in `docs/observability-validation.md`

CI remains the final remote confirmation for this milestone after the branch is pushed.

### P3-2 Backend <-> Core-Agent Trust Boundary

In progress:

- Token auth mode is implemented for backend -> core-agent gRPC metadata.
- Core-agent rejects missing or wrong `x-snowpanel-agent-token` metadata in token mode.
- Local `none` mode remains available for development.
- `mtls` config fields are reserved and fail fast until certificate support is implemented.
- Backend maps gRPC `Unauthenticated` to `core agent authentication failed`.
- Tests cover token metadata injection, auth failure mapping, and core-agent allow/deny paths without leaking token values.
- Local token-auth compose smoke passed for `/health` and `/ready`.

### P3-3 Secrets & Settings Hardening

Local P3-3 secrets and settings hardening is complete:

- Audit request summaries, audit result messages, and task log metadata are redacted before persistence for common sensitive keys.
- Frontend token storage decision is documented in `docs/security-token-storage-decision.md`.
- AES-256-GCM encryption helpers and `SNOWPANEL_ENCRYPTION_KEY` config are implemented.
- SystemSetting repository/service encryption wiring is implemented for `is_encrypted=true` values.
- Startup validation rejects invalid encryption keys and fails fast when encrypted settings exist without a configured key.
- Local gates passed: `go test ./...`, `make lint`, and `make test`.

CI remains the final remote confirmation for this milestone after the branch is pushed.

### P3-4 Durable Task Worker

Local P3-4 durable worker wiring is in progress:

- Durable task schema fields are added through `backend/migrations/0002_durable_tasks.*.sql`.
- Task model now carries lease, retry, idempotency, and next-run metadata.
- Worker configuration envs are available under `TASK_WORKER_*`.
- Task repository exposes claim, heartbeat, complete, fail, and stale-release APIs.
- Backend startup now runs the DB-backed worker when `TASK_WORKER_ENABLED=true`.
- Task creation queues work for the durable worker by default; the legacy goroutine executor remains available when the worker is disabled.
- Docker/service restart task creation accepts an optional `idempotency_key`; duplicate keys return the existing task instead of enqueueing duplicate work.
- The worker loop supports concurrency, lease heartbeat, stale lease release, max-attempt retry, retry backoff, and context-based graceful shutdown.
- Task and worker metrics now expose queue depth, running count, completion totals, task duration, and claim outcomes.
- Service tests cover enqueue-only mode, idempotency duplicate handling, worker claim/execute, worker context cancellation, claim-only-once behavior, stale lease recovery, failed task retry, max-attempt exhaustion, queued/running cancellation, long-running heartbeat behavior, and task metric recording.
- Local backend gate passed: `cd backend && go test ./...`.

## Follow-up Hardening (Post-P3-0)

1. Wire final alert destinations to real on-call channels under team policy
2. Tune dedup/escalation windows and SLO thresholds against production traffic
3. Add browser/frontend tracing if future troubleshooting depth requires it
4. Implement backend <-> core-agent authentication for the production trust boundary
5. Add secrets-at-rest encryption and production startup validation
6. Replace goroutine-only async tasks with a durable DB-backed worker

## Not Current Priorities

- UI polish or visual redesign
- New pages before operational gaps are closed
- Premature “fully production-ready” positioning before operational governance is finalized
