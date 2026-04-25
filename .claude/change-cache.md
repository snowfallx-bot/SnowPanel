【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮目标：按用户“加快推进 P2-2”的要求，优先补齐可观测性生产化缺口中最可落地的部分（SLO/告警分层/通知模板），并保持小步快提交。

本轮实际改动

1. Prometheus 告警规则升级为 SLO 分层基线
   - 文件：`deploy/observability/prometheus/alerts/snowpanel-alerts.yml`
   - 新增 recording rules：
     - `snowpanel:backend_http_total:rate5m`
     - `snowpanel:backend_http_5xx:rate5m`
     - `snowpanel:backend_http_availability:ratio5m`
     - `snowpanel:core_agent_grpc_error_ratio:ratio5m`
   - 新增 critical 分级：
     - `SnowPanelBackendP95LatencyCritical`
     - `SnowPanelCoreAgentP95LatencyCritical`
     - `SnowPanelCoreAgentGrpcErrorRateCritical`
   - 新增 backend availability SLO 告警：
     - `SnowPanelBackendAvailabilitySLOWarning`
     - `SnowPanelBackendAvailabilitySLOCritical`
   - 并将 core-agent gRPC warning 阈值收敛为更早预警（2%），critical 维持高阈值（5%）。

2. Alertmanager 基线路由从“单 no-op”升级为 warning/critical 双通道骨架
   - 文件：`deploy/observability/alertmanager/alertmanager.yml`
   - 路由改为：
     - `severity="warning"` -> `snowpanel-warning`
     - `severity="critical"` -> `snowpanel-critical`
   - 两个 receiver 均保留 no-op 注释模板，便于后续直接接 webhook/email/slack/wechat。

3. 新增生产接收器模板文件（加速真实通知落地）
   - 文件：`deploy/observability/alertmanager/alertmanager.production.example.yml`（新增）
   - 提供 warning/critical 双 receiver 的可执行模板结构，可作为生产配置起点。

4. 文档与路线图同步
   - 更新：
     - `docs/observability.md`
     - `docs/observability.zh-CN.md`
     - `docs/roadmap.md`
     - `docs/roadmap.zh-CN.md`
     - `.claude/progress.md`
   - 内容同步：
     - SLO recording rules 与新告警项
     - Alertmanager warning/critical 双通道路由说明
     - 生产模板文件入口

本轮本地验证

1. 能做的：
   - 配置与文档引用一致性扫描（`rg`）通过。

2. 受限项：
   - 当前环境无 `docker`、无 `python`、无 `promtool`，无法在本机做 Prometheus/Alertmanager 真实加载验证。
   - 因此本轮主要完成“配置落地 + 文档一致性 + 可执行模板”，实跑验证留给有环境机器。

commit 摘要

- `f35455b feat(observability): add slo recording rules and severity routing baseline`
- `ed42111 docs(observability): add production alertmanager receiver template`

希望接下来的 AI 做什么

1. 在具备 Docker 的环境优先做 P2-2 实跑验收（非常关键）
   - 启动：
     - `make up-observability` 或 `make up-host-agent-observability`
   - 验证：
     - `pwsh -File ./scripts/ci/observability-smoke.ps1`
     - 或手动触发 GitHub Actions `Observability Smoke` workflow
   - 检查点：
     - Prometheus 规则加载成功（含新 recording/alerts）
     - Alertmanager warning/critical 路由生效
     - Jaeger 跨服务 trace 串联通过

2. 立即推进真实通知渠道接入（P2-2 最后关键缺口）
   - 基于 `alertmanager.production.example.yml` 填入真实 webhook/email/slack/wechat
   - 结合 `scripts/observability/alertmanager-smoke.ps1` 做投递验收

3. 继续小步清理 P2-3
   - 重点扫尾非主文档、注释与 fixture 的历史措辞

by: gpt-5.5
