【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续推进 `P2-2`，把可观测性从“可采指标”推进到“可落地报警”。

本次核心判断

1. 目前 backend/core-agent 都有 metrics，但缺统一抓取入口和默认告警规则，现场排障仍依赖人工看图。
2. 在 OTel 大改前，先落 Prometheus 基线 stack 与 alert rules，能最快形成生产可执行的观测闭环。

本轮实际改动

1. 新增 observability compose 覆盖
   - `docker-compose.observability.yml`
   - 提供 `prometheus` 服务、持久卷、端口映射（默认 `9090`），并支持抓取 host-agent（`host.docker.internal`）。

2. 新增 Prometheus 基线配置与告警
   - `deploy/observability/prometheus/prometheus.yml`
   - `deploy/observability/prometheus/alerts/snowpanel-alerts.yml`
   - 默认抓取目标：
     - `snowpanel-backend` -> `backend:8080/metrics`
     - `snowpanel-core-agent-compose` -> `core-agent:9108/metrics`
     - `snowpanel-core-agent-host` -> `host.docker.internal:9108/metrics`
   - 基线告警：
     - `SnowPanelBackendDown`
     - `SnowPanelCoreAgentMetricsDown`
     - `SnowPanelBackendP95LatencyHigh`
     - `SnowPanelCoreAgentP95LatencyHigh`
     - `SnowPanelBackendAgentTransportErrorsHigh`
     - `SnowPanelCoreAgentGrpcErrorRateHigh`
     - `SnowPanelCoreAgentInFlightHigh`

3. Makefile 增加 observability 相关目标
   - `up-observability` / `down-observability` / `logs-observability`
   - `up-host-agent-observability` / `down-host-agent-observability` / `logs-host-agent-observability`

4. 配置与文档同步
   - `.env.example` 增加 `PROMETHEUS_PORT=9090`
   - 更新文档：
     - `docs/observability.md` / `docs/observability.zh-CN.md`
     - `docs/deployment.md` / `docs/deployment.zh-CN.md`
   - 更新进度状态：
     - `progress.md` 标注“Prometheus 基线部署 + 基线告警规则”已完成，`P2-2` 仍进行中（OTel/Alertmanager/SLO 阈值校准未完成）。

本轮修改文件

- `.claude/change-cache.md`
- `.claude/progress.md`
- `.env.example`
- `Makefile`
- `docker-compose.observability.yml`
- `deploy/observability/prometheus/prometheus.yml`
- `deploy/observability/prometheus/alerts/snowpanel-alerts.yml`
- `docs/observability.md`
- `docs/observability.zh-CN.md`
- `docs/deployment.md`
- `docs/deployment.zh-CN.md`

本地验证

- 已通过：
  - `go test ./internal/grpcclient ./internal/middleware ./internal/api`
- 未做：
  - docker compose 实际启动验证（当前环境无 docker）
  - rust 侧编译验证（当前环境无 cargo）

commit摘要

- 计划提交：`feat(observability): add prometheus baseline stack and alert rules`

希望接下来的 AI 做什么

1. 在具备 docker 的环境启动：
   - `make up-observability` 或 `make up-host-agent-observability`
   - 验证 Prometheus targets 与 alerts 载入状态。
2. 根据真实运行数据校准告警阈值（p95、错误率、in-flight）。
3. 继续 `P2-2` 剩余项：
   - 接入 Alertmanager（通知路由）
   - 设计 OTel collector/exporter（backend + core-agent 统一）

by: gpt-5.5
