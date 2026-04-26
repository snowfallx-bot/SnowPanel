【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============
本轮继续按“加快 P2-2 收口”推进，重点从“脚本可跑”升级到“host-agent 模式可在 workflow 里直接执行”。

本轮实际改动

1. host-agent 观测冒烟工作流真正可执行
   - `.github/workflows/observability-smoke.yml`
   - 在 `agent_mode=host-agent` 时新增完整链路：
     - 安装 Rust toolchain
     - 构建 `core-agent` release 二进制
     - 在 runner 宿主机后台启动 `core-agent`
     - 轮询 `127.0.0.1:50051` 端口直到 ready
     - 失败时上传 host-agent 日志 artifact
     - 结束时回收 host-agent 进程
   - workflow 新增输入：
     - `host_agent_metrics_base_url`
   - 运行 smoke 脚本时透传：
     - `agent_mode`
     - `host_agent_target`
     - `host_agent_metrics_base_url`

2. host-agent 模式下的 CI 脚本前置探测
   - `scripts/ci/observability-smoke.ps1`
   - 新增参数 `-HostAgentMetricsBaseUrl`（默认 `http://127.0.0.1:9108`）
   - 在 `host-agent` 模式下，先探测宿主机 `core-agent` `/metrics` 可用且包含关键指标，再继续拉起 backend/observability 栈。

3. warning 告警冒烟去固定实例名
   - `scripts/ci/observability-smoke.ps1`
   - 调用 `alertmanager-smoke.ps1` 时不再传固定 `-Instance`，直接复用其唯一默认实例，避免历史同名告警干扰。

4. tracing 冒烟强关联校验（本轮起始已并入）
   - `scripts/observability/common.ps1`
     - 新增 `Invoke-ObservabilityApiRequest`，统一返回 `status/content/json/headers`
   - `scripts/observability/trace-smoke.ps1`
     - 强校验响应 `X-Request-ID`
     - 校验 Jaeger 中同一 request_id 的 backend/core-agent 关联
     - 校验 core-agent 必含两个关键 `grpc.method` span

本轮本地验证

1. 已执行并通过：
   - PowerShell 语法解析：
     - `scripts/ci/observability-smoke.ps1`
     - `scripts/observability/common.ps1`
     - `scripts/observability/trace-smoke.ps1`

2. 说明：
   - 当前环境未实跑 Docker + host-agent 全链路；运行态验收需在具备 Docker 的环境执行 workflow 或脚本实跑。

commit 摘要

- `819bf19 test(observability): tighten trace correlation validation with request id`
- `866535d ci(observability): bootstrap host core-agent for host-mode smoke runs`
- `e267437 test(ci): preflight host-agent metrics in observability smoke`
- `0fa9da9 ci(observability): expose host-agent metrics input for smoke workflow`
- `75c948e test(ci): let warning alert smoke use unique default instance`

希望接下来的 AI 做什么

1. 优先执行运行态验收（P2-2 最关键）：
   - 手动 workflow 触发两次：
     - `agent_mode=container-agent`
     - `agent_mode=host-agent`
   - 确认两次都通过 trace + alert 双冒烟。

2. 若 host-agent 模式失败，优先排查：
   - workflow 中 host-agent 是否在 smoke 前成功监听 `50051`
   - `host_agent_target` 与 backend 实际连通是否一致
   - Jaeger 中 core-agent span 是否带 `snowpanel.request_id` 与必需 `grpc.method`

3. 验收通过后继续 P2-3（仅代码）：
   - 继续清理脚本中重复 wait/request 逻辑，不做文档修补。

by: gpt-5.5
