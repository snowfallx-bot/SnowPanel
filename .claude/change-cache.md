【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“加快 P2-2”推进，目标是把 observability 配置闸门从“语法校验”升级为“行为校验”。

本轮实际改动

1. 新增 Prometheus 告警规则单测文件
   - `deploy/observability/prometheus/tests/snowpanel-alerts.test.yml`（新增）
   - 覆盖关键 critical 告警触发路径：
     - `SnowPanelBackendDown`
     - `SnowPanelCoreAgentMetricsDown`
     - `SnowPanelBackendAvailabilitySLOCritical`
     - `SnowPanelCoreAgentGrpcErrorRateCritical`

2. 将 `promtool test rules` 接入配置校验脚本
   - `scripts/observability/validate-config.ps1`
   - 在原有 `promtool check config` / `promtool check rules` / `amtool check-config` 基础上，新增：
     - `promtool test rules /etc/prometheus/tests/snowpanel-alerts.test.yml`

3. 中英文文档与 roadmap 同步
   - `scripts/observability/README.md`
   - `scripts/ci/README.md`
   - `docs/observability.md` / `docs/observability.zh-CN.md`
   - `docs/development.md` / `docs/development.zh-CN.md`
   - `docs/roadmap.md` / `docs/roadmap.zh-CN.md`
   - `.claude/progress.md`
   - 统一说明：`validate-config.ps1` 已包含告警规则单测闸门（不仅是配置语法检查）。

本轮本地验证

1. 已执行：
   - PowerShell 语法解析校验 `scripts/observability/validate-config.ps1`

2. 结果：
   - 语法通过。

3. 环境限制：
   - 当前机器无 `docker`，无法本地执行容器内 `promtool test rules` 的真实运行校验；
   - 需在具备 Docker 的环境（或 CI）完成该链路验收。

commit 摘要

- `b8b9a39 feat(observability): add promtool alert rule unit tests`
- `3d30dd7 docs(observability): document promtool alert rule test gate`

希望接下来的 AI 做什么

1. 在具备 Docker 的环境跑通 observability 闸门
   - `pwsh -File ./scripts/observability/validate-config.ps1`
   - 确认 `promtool test rules` 通过

2. 继续收口 P2-2 剩余项
   - 将 `alertmanager.production.example.yml` 替换为真实 warning/critical 接收器配置
   - 用 `pwsh -File ./scripts/observability/alertmanager-smoke.ps1` 做真实通知通道验收
   - 在 compose / host-agent 两种模式完成 tracing 端到端实测留痕

by: gpt-5.5
