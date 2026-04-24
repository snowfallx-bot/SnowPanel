【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续推进 `P2-2`（生产可观测性），重点完成了“请求链路串联 + 指标维度增强 + 可观测文档落地”。

本次核心判断

1. backend 侧已有 `/metrics`，但 agent RPC 指标缺少按 RPC 维度拆分，排障时不够直观。
2. backend request-id 只在 HTTP 层可见，未进入 gRPC metadata，导致 backend 与 core-agent 的日志无法稳定按同一请求串联。
3. 在不引入重型 OTel 基建前，先把 `X-Request-ID` 链路打通并固化排障手册，是当前最稳妥、收益最高的改动。

本轮实际改动

1. backend request-id 上下文透传
   - 新增 `backend/internal/requestctx/request_id.go`，统一 request-id 的 context 存取。
   - `RequestID` middleware 现会把 request-id 写入 `c.Request.Context()`，不再只停留在 gin context。

2. backend -> core-agent gRPC metadata 透传
   - `backend/internal/grpcclient/agent_client.go`：
     - 在 `invoke()` 内从 context 读取 request-id，并注入 gRPC metadata `x-request-id`。
     - 新增 `requestIDMetadataKey` 常量与 `withOutgoingRequestID()`。

3. agent RPC 指标维度增强
   - `backend/internal/metrics/metrics.go`：
     - `snowpanel_agent_requests_total` 与 `snowpanel_agent_request_duration_seconds` 增加 `rpc` 标签。
   - `backend/internal/grpcclient/agent_client.go`：
     - `invoke()` 新增 `rpcName` 入参。
     - 各 RPC 调用点都标注了固定 rpc 名称（如 `health.check`、`file.list`、`service.restart`、`docker.list_containers` 等）。

4. core-agent 日志接入 request-id
   - `core-agent/src/api/grpc_server.rs`：
     - 所有 gRPC service 统一挂载 interceptor。
     - interceptor 读取 metadata `x-request-id`，记录 `request_id` + `grpc_method` 日志字段，实现跨 backend/agent 日志联查。

5. 测试补齐
   - 新增 `backend/internal/middleware/request_id_test.go`：验证 request-id 已写入请求 context。
   - `backend/internal/grpcclient/agent_client_contract_test.go` 新增用例：验证 gRPC metadata request-id 透传。
   - `backend/internal/grpcclient/agent_client_metrics_test.go` 更新：校验 agent 指标新标签维度（含 rpc）。

6. 文档落地
   - 新增：
     - `docs/observability.md`
     - `docs/observability.zh-CN.md`
   - 内容包括：指标说明、request-id 串联路径、快速排障流程、当前缺口。
   - `progress.md` 的 `P2-2` 已更新为“进行中”，并同步新增能力与剩余缺口。

本轮修改文件

- `.claude/change-cache.md`
- `.claude/progress.md`
- `backend/internal/requestctx/request_id.go`
- `backend/internal/middleware/request_id.go`
- `backend/internal/middleware/request_id_test.go`
- `backend/internal/metrics/metrics.go`
- `backend/internal/grpcclient/agent_client.go`
- `backend/internal/grpcclient/agent_client_contract_test.go`
- `backend/internal/grpcclient/agent_client_metrics_test.go`
- `core-agent/src/api/grpc_server.rs`
- `docs/observability.md`
- `docs/observability.zh-CN.md`

本地验证

- 已通过：
  - `go test ./internal/grpcclient ./internal/middleware ./internal/api`
  - `go test ./internal/requestctx`
- 当前环境缺少 `cargo`，无法本地执行 `core-agent` 的 `cargo fmt/cargo test`，该部分需依赖 CI 验证。

commit摘要

- 计划提交：`feat(observability): propagate request id across backend and core-agent`

希望接下来的 AI 做什么

1. 观察 CI 中 core-agent 编译/测试是否通过（重点关注 `with_interceptor` 兼容性）。
2. 若通过，继续推进 `P2-2` 剩余项：
   - 评估是否为 core-agent 增加独立 `/metrics` 暴露。
   - 设计 OTel/trace backend（Jaeger/Tempo）最小接入方案。
3. 若 CI 报错，优先修正 core-agent interceptor 相关兼容问题，再继续 P2-2。

by: gpt-5.5
