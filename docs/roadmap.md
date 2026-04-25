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

## In Progress

### P2-2 Production Observability

- Prometheus metrics and baseline alert rules are in place
- Alertmanager baseline routing is in place
- OTEL tracing baseline now exists for:
  - backend HTTP spans
  - backend gRPC client spans
  - core-agent gRPC server spans
  - OTel Collector -> Jaeger pipeline
- Observability smoke scripts now exist for:
  - Jaeger cross-service trace validation (`scripts/observability/trace-smoke.ps1`)
  - Alertmanager synthetic alert injection validation (`scripts/observability/alertmanager-smoke.ps1`)
  - One-shot trace + alertmanager validation (`scripts/observability/full-smoke.ps1`)

Still remaining:

- Validate tracing end to end in compose and host-agent runtime modes
- Wire real alert notification channels
- Tune alert deduplication, escalation, and SLO/SLI thresholds

### P2-3 Prototype-Trace Cleanup

- Outdated backend README legacy prototype notes have been removed
- App shell and frontend e2e marker no longer use `Linux Panel Prototype`
- Root README now exposes observability commands and docs
- Some historical wording and duplicate docs still need cleanup

## Next Priorities

1. Validate the tracing pipeline in a Docker-capable environment
2. Finish production-ready alert delivery and threshold tuning
3. Continue pruning outdated docs, legacy wording, and duplicated explanations

## Not Current Priorities

- UI polish or visual redesign
- New pages before operational gaps are closed
- Marketing-style “production ready” positioning
