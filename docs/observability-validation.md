# Observability Validation Records

Language: **English** | [简体中文](observability-validation.zh-CN.md)

This file tracks concrete end-to-end observability validation evidence in CI.

## Latest Verified Run

- Workflow run: `24971113137`
- Run URL: <https://github.com/snowfallx-bot/SnowPanel/actions/runs/24971113137>
- Trigger: `push` to `main`
- Head commit: `436781fcd25aaa6c16f7a449f6011e21d8a7e64d`
- Created at: `2026-04-27 00:35:19` (Asia/Shanghai)
- Conclusion: `success`

## Required Jobs

The following jobs all completed with `success` in this run:

- `observability-config`
- `compose-smoke`
- `observability-smoke-container`
- `observability-smoke-host-agent`
- `backend-integration`
- `frontend-e2e`

## Interpretation

- Compose-mode observability pipeline is verified by `observability-smoke-container`.
- Host-agent observability pipeline is verified by `observability-smoke-host-agent`.
- Mainline auth/session/files/tasks/agent contracts remain healthy in the same run.

## Local P3-1 Alerting Validation

- Date: `2026-05-10` (Asia/Shanghai)
- Branch: `main`
- Scope: Alertmanager production config generation, config validation, warning/critical synthetic routing, and inhibition smoke.

Commands and results:

- `pwsh -File ./scripts/observability/generate-alertmanager-config.ps1 ... -OutputPath deploy/observability/alertmanager/alertmanager.generated.smoke.yml`: passed
- `pwsh -File ./scripts/observability/validate-config.ps1 -ExtraAlertmanagerConfigFiles deploy/observability/alertmanager/alertmanager.generated.smoke.yml`: passed
- `pwsh -File ./scripts/observability/alertmanager-smoke.ps1 -Severity warning -AlertName SnowPanelP31SmokeWarning`: passed, receiver `snowpanel-warning`
- `pwsh -File ./scripts/observability/alertmanager-smoke.ps1 -Severity critical -AlertName SnowPanelP31SmokeCritical`: passed, receiver `snowpanel-critical`
- `pwsh -File ./scripts/observability/alertmanager-inhibition-smoke.ps1 -AlertName SnowPanelP31InhibitionSmoke`: passed

Fixed during validation:

- The warning-only Prometheus alert rule fixture used a backend 5xx ratio high enough to trigger the burn-rate critical alert. It now uses a warning-only ratio so warning and critical routing remain distinct.
