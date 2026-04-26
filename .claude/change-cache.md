【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“加快 P2-2”推进，重点把 observability 配置闸门升级为“含规则行为断言”的可回归能力。

本轮实际改动

1. 新增并接入 Prometheus 规则单测
   - `deploy/observability/prometheus/tests/snowpanel-alerts.test.yml`（新增）
   - `scripts/observability/validate-config.ps1`（更新）
   - `validate-config.ps1` 现已执行：
     - `promtool check config`
     - `promtool check rules`
     - `promtool test rules /etc/prometheus/tests/snowpanel-alerts.test.yml`
     - `amtool check-config`

2. 扩展告警分级回归覆盖
   - `snowpanel-alerts.test.yml` 新增 warning-only 场景：
     - backend availability warning 触发、critical 不触发
     - core-agent grpc error warning 触发、critical 不触发
   - 与已有 critical 场景组合后，覆盖“分级触发/不升级”两类关键断言。

3. 文档与路线图同步
   - `scripts/observability/README.md`
   - `scripts/ci/README.md`
   - `docs/observability.md` / `docs/observability.zh-CN.md`
   - `docs/development.md` / `docs/development.zh-CN.md`
   - `docs/roadmap.md` / `docs/roadmap.zh-CN.md`
   - `.claude/progress.md`
   - 统一注明：`validate-config.ps1` 不再只是语法检查，也包含 `promtool test rules` 的规则行为校验。

本轮本地验证

1. 已执行：
   - PowerShell 语法解析校验 `scripts/observability/validate-config.ps1`

2. 结果：
   - 语法通过。

3. 环境限制：
   - 当前机器无 `docker`，无法本地执行容器内 `promtool test rules` 实跑；
   - 需在 CI 或 Docker 环境完成端到端验收。

commit 摘要

- `b8b9a39 feat(observability): add promtool alert rule unit tests`
- `3d30dd7 docs(observability): document promtool alert rule test gate`
- `e03bcbf test(observability): expand promtool alert rule scenarios`

希望接下来的 AI 做什么

1. 在有 Docker 的环境执行 P2-2 校验链路
   - `pwsh -File ./scripts/observability/validate-config.ps1`
   - `pwsh -File ./scripts/ci/observability-smoke.ps1`
   - 确认新加 warning-only/critical 场景均通过

2. 继续收口 P2-2 剩余项
   - 基于 `alertmanager.production.example.yml` 接入真实 warning/critical receiver
   - 用 `alertmanager-smoke.ps1` 做真实通知通道验收
   - 在 compose / host-agent 两种模式补齐 tracing 端到端实测留痕

by: gpt-5.5
