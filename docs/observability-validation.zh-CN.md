# 可观测性实测记录

语言: [English](observability-validation.md) | **简体中文**

该文档用于沉淀 CI 中可观测性端到端实测的可追溯证据。

## 最新通过记录

- Workflow run：`24971113137`
- Run 链接：<https://github.com/snowfallx-bot/SnowPanel/actions/runs/24971113137>
- 触发方式：`push` 到 `main`
- 对应提交：`436781fcd25aaa6c16f7a449f6011e21d8a7e64d`
- 创建时间：`2026-04-27 00:35:19`（Asia/Shanghai）
- 结果：`success`

## 关键任务通过项

该 run 中以下关键任务均为 `success`：

- `observability-config`
- `compose-smoke`
- `observability-smoke-container`
- `observability-smoke-host-agent`
- `backend-integration`
- `frontend-e2e`

## 结论

- `observability-smoke-container` 证明 compose 模式可观测性链路通过。
- `observability-smoke-host-agent` 证明 host-agent 模式可观测性链路通过。
- 同一 run 下主链路（认证/会话、文件、任务、agent 契约）也保持健康。
