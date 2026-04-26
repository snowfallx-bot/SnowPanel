【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============
本轮继续按“加快 P2-2 收口”为主线推进，保持只改代码与脚本，不修补业务文档。

本轮实际改动

1. tracing 冒烟校验升级为“强关联”
   - `scripts/observability/common.ps1`
     - 新增 `Invoke-ObservabilityApiRequest`，统一返回 `status/content/json/headers`。
   - `scripts/observability/trace-smoke.ps1`
     - 触发 dashboard 后强校验响应 `X-Request-ID` 与请求一致；
     - Jaeger 轮询从“仅两服务出现”升级为“request_id 精确关联”：
       - backend span 必须带 `snowpanel.request_id=<request_id>`
       - core-agent span 必须带同一 `snowpanel.request_id`
       - core-agent `grpc.method` 必须覆盖：
         - `/snowpanel.agent.v1.SystemService/GetSystemOverview`
         - `/snowpanel.agent.v1.SystemService/GetRealtimeResource`
     - 失败时输出最近观测信息，便于定位。

2. observability-smoke 支持 host-agent 模式
   - `scripts/ci/observability-smoke.ps1`
   - 新增参数：
     - `-AgentMode container-agent|host-agent`（默认 container-agent）
     - `-HostAgentTarget`（默认 `host.docker.internal:50051`）
   - host-agent 模式下自动叠加 `docker-compose.host-agent.yml`，并按模式切换起服务集合（container 模式起 `core-agent`，host 模式不拉起容器内 agent）。

3. 手动 workflow 支持选择 agent 模式
   - `.github/workflows/observability-smoke.yml`
   - `workflow_dispatch` 新增输入：
     - `agent_mode`
     - `host_agent_target`
   - 调用 smoke 脚本时透传上述参数。

4. Alertmanager 冒烟进一步降误判
   - `scripts/observability/alertmanager-smoke.ps1`
   - 默认 `Instance` 改为运行时唯一值（时间戳后缀），避免历史残留同名告警干扰；
   - 查询 filter 从单一 `alertname` 扩展为 `alertname + instance + severity`（同时用于 `/alerts` 与 `/alerts/groups`），降低误命中概率。

本轮本地验证

1. 已执行并通过：
   - PowerShell 语法解析：
     - `scripts/observability/common.ps1`
     - `scripts/observability/trace-smoke.ps1`
     - `scripts/observability/alertmanager-smoke.ps1`
     - `scripts/ci/observability-smoke.ps1`

2. 说明：
   - 当前环境未实跑 Docker 链路；`compose/host-agent` 运行态验收需在具备 Docker 且可访问 host-agent 的环境执行。

commit 摘要

- `819bf19 test(observability): tighten trace correlation validation with request id`
- `244d2e7 feat(ci): support host-agent mode in observability smoke script`
- `cf375ca ci(observability): add host-agent mode inputs for smoke workflow`
- `442fc5a test(observability): harden alert smoke filters and unique default instance`

希望接下来的 AI 做什么

1. 在可执行环境做 P2-2 运行态验收（优先）：
   - container-agent：
     - `pwsh -File ./scripts/ci/observability-smoke.ps1 -AgentMode container-agent`
   - host-agent：
     - `pwsh -File ./scripts/ci/observability-smoke.ps1 -AgentMode host-agent -HostAgentTarget <host-agent:50051>`
   - 重点确认 trace request_id 关联与 warning/critical 路由校验均通过。

2. 若 host-agent 模式失败，优先定位：
   - host-agent OTEL 导出端点是否指向可达 collector（通常主机 `127.0.0.1:4317`）；
   - backend `AGENT_TARGET` 与 host-agent 监听地址是否一致；
   - Jaeger 中 core-agent span 是否包含 `snowpanel.request_id` 与 `grpc.method`。

3. 验收通过后，再继续 P2-3 代码清理：
   - 优先清理脚本与前端页面的重复查询键、重复错误处理和重复请求包装逻辑（仍不动文档美化）。

by: gpt-5.5
