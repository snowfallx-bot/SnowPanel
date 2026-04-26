【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============
本轮继续按“加快 P2-2 收口”推进，核心是把 observability 冒烟链路再收敛一层并提高 host-agent 模式的执行稳健性。

本轮实际改动

1. full-smoke 统一双严重级别告警校验入口
   - `scripts/observability/full-smoke.ps1`
   - 新增 `-ValidateAllAlertSeverities` 开关：
     - 开启后会在一次 full-smoke 中依次校验 `critical` + `warning`
     - 自动为不同严重级别生成区分的告警名后缀，避免互相干扰
   - 保留原有默认行为（不加开关时仍只校验单一 severity）。

2. CI observability 冒烟去掉重复 warning 调用
   - `scripts/ci/observability-smoke.ps1`
   - 改为只调用一次 `full-smoke.ps1 -ValidateAllAlertSeverities`，删除原先重复的单独 warning 调用段。
   - 同时补强执行稳健性：
     - `container-agent` 模式显式启用 compose profile：`--profile container-agent`
     - `container-agent` 模式下主动清理 `AGENT_TARGET`，避免环境串扰误连 host-agent
     - `host-agent` 模式下新增 `HostAgentTarget` 非空校验
     - `HostAgentMetricsBaseUrl` 支持 base URL / 完整 metrics URL 两种输入并自动归一化，避免 `.../metrics/metrics` 误拼接

3. host-agent workflow 端口/指标地址与输入对齐
   - `.github/workflows/observability-smoke.yml`
   - 在启动宿主机 core-agent 时，不再把端口写死为 `50051`：
     - 从 `host_agent_target` 解析端口并用于 `CORE_AGENT_PORT` 与就绪探测
     - 从 `host_agent_metrics_base_url` 解析端口并用于 `CORE_AGENT_METRICS_PORT`
   - 避免“输入改了，但宿主机 core-agent 仍按默认端口启动”的不一致问题。

本轮本地验证

1. 已执行并通过：
   - PowerShell 语法解析：
     - `scripts/observability/full-smoke.ps1`
     - `scripts/ci/observability-smoke.ps1`

2. 说明：
   - 当前环境未实跑 Docker/Actions；运行态验收需在 GitHub workflow 或具备 Docker 的环境执行。

commit 摘要

- `fbc9751 refactor(observability): unify dual-severity alert checks via full smoke`
- `02b1d90 test(ci): harden observability smoke env isolation and profile selection`
- `639583f ci(observability): align host-agent bootstrap ports with workflow inputs`
- `be0c259 test(ci): normalize host-agent metrics endpoint input`

希望接下来的 AI 做什么

1. 直接触发 `Observability Smoke` workflow 两次做验收：
   - `agent_mode=container-agent`
   - `agent_mode=host-agent`
   - 关注是否都通过 trace 强关联 + alert 双级别校验。

2. 若 host-agent 模式失败，优先检查：
   - workflow 日志里解析出的 `host_agent_target` 端口是否与 backend 连接目标一致
   - host-agent 启动日志是否有 OTLP 导出错误（collector 端口不可达）
   - Jaeger 中是否出现同 request_id 的 backend/core-agent span 以及必需 grpc.method。

3. 验收通过后继续 P2-3（仅代码）：
   - 继续收敛脚本重复逻辑，不补文档。

by: gpt-5.5
