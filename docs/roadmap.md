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

## Follow-up Hardening (Post-P2, Non-blocking)

1. Wire final alert destinations to real on-call channels under team policy
2. Tune dedup/escalation windows and SLO thresholds against production traffic
3. Add browser/frontend tracing if future troubleshooting depth requires it

## Not Current Priorities

- UI polish or visual redesign
- New pages before operational gaps are closed
- Premature “fully production-ready” positioning before operational governance is finalized
