【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续按“加快 P2-2”推进，重点补上“规则已加载”的自动校验，避免只检查配置语法但漏掉运行时规则注册问题。

本轮实际改动

1. 新增 Prometheus 规则加载冒烟脚本
   - `scripts/observability/prometheus-rules-smoke.ps1`（新增）
   - 对运行中的 `/api/v1/rules` 做检查：
     - 校验关键 recording rules 是否存在
     - 校验关键 alert rules 是否存在
   - 覆盖本项目当前 SLO/分级告警基线（availability、error ratio、latency critical/warning 等）。

2. 将规则加载校验接入 observability 冒烟主流程
   - `scripts/ci/observability-smoke.ps1`
   - 新增：
     - Prometheus rules API 就绪等待
     - 调用 `prometheus-rules-smoke.ps1` 做规则存在性校验
   - 顺序变为：配置校验 -> 栈启动 -> 规则加载校验 -> trace/alertmanager 冒烟。

3. 文档入口同步
   - `scripts/observability/README.md`
   - `scripts/ci/README.md`
   - `docs/development.md`
   - `docs/development.zh-CN.md`
   - `docs/observability.md`
   - `docs/observability.zh-CN.md`
   - `.claude/progress.md`
   - 新增脚本命令说明，并标注其在 observability 冒烟链路中的作用。

本轮本地验证

1. 已执行：
   - `pwsh -File ./scripts/observability/prometheus-rules-smoke.ps1 -PrometheusBaseUrl http://127.0.0.1:1`

2. 结果：
   - 在不可达地址下按预期网络失败，说明脚本启动与失败路径正常。

3. 环境限制：
   - 当前机器仍无 `docker`，无法在本地跑通真实 `prometheus/api/v1/rules` 在线校验；
   - 需在具备 Docker 的环境完成端到端验证。

commit 摘要

- `5165417 feat(observability): add prometheus rules smoke validation`

希望接下来的 AI 做什么

1. 在有 Docker 的环境跑完整 P2-2 验收链路
   - `pwsh -File ./scripts/observability/validate-config.ps1`
   - `pwsh -File ./scripts/ci/observability-smoke.ps1`
   - 确认 `prometheus-rules-smoke` 通过（规则全部存在）

2. 接入真实通知通道并验证
   - 以 `alertmanager.production.example.yml` 为模板
   - 接入 warning/critical 实际 receiver
   - 使用 `alertmanager-smoke.ps1` 做投递验收

3. 然后继续清理 P2-3 尾项
   - 聚焦非主文档与注释中的历史措辞

by: gpt-5.5
