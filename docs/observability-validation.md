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
