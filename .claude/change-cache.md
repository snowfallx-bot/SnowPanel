【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续推进 `P2-2`，把上轮“链路串联”再向前推进成“core-agent 独立指标端点 + gRPC 方法级指标”。

本次核心判断

1. 上轮已打通 request-id 到 core-agent，但仍缺 core-agent 自身可抓取指标端点，排障仍偏依赖日志。
2. 现阶段先补 Prometheus 端点和方法级 gRPC 指标，能在不引入 OTel 大改的前提下显著提升定位效率。
3. 由于本地环境无 `cargo`，Rust 编译正确性需要交给 CI 再确认。

本轮实际改动

1. core-agent 新增 metrics server（独立 HTTP 端点）
   - 新增模块：
     - `core-agent/src/observability/mod.rs`
     - `core-agent/src/observability/metrics.rs`
   - 使用 Prometheus 默认 registry 暴露 `/metrics`。
   - 新增指标：
     - `snowpanel_core_agent_grpc_requests_total{grpc_method,outcome}`
     - `snowpanel_core_agent_grpc_request_duration_seconds{grpc_method,outcome}`
     - `snowpanel_core_agent_grpc_requests_in_flight{grpc_method}`

2. core-agent gRPC 请求接入指标采集
   - `core-agent/src/api/grpc_server.rs`：
     - 增加 `observe_grpc_call()` 包装器。
     - 关键 gRPC handler（health/system/files/services/docker/cron）都接入方法级计数与时延采集。
     - 保留上轮 request-id 日志拦截逻辑。

3. core-agent 启动流程支持并发运行 gRPC + metrics
   - `core-agent/src/main.rs`：
     - 新增 `mod observability`。
     - `CORE_AGENT_METRICS_ENABLED=true` 时，`tokio::try_join!` 并发启动 gRPC server 与 metrics server。

4. core-agent 配置项扩展
   - `core-agent/src/config/mod.rs` 新增：
     - `CORE_AGENT_METRICS_ENABLED`（默认 `true`）
     - `CORE_AGENT_METRICS_HOST`（默认 `127.0.0.1`）
     - `CORE_AGENT_METRICS_PORT`（默认 `9108`）
   - 新增 `metrics_address()`。

5. 运行时配置与文档同步
   - `core-agent/Cargo.toml` 增加依赖：`axum`、`prometheus`、`once_cell`。
   - `.env.example` 增加 core-agent metrics 配置项。
   - `docker-compose.yml` 为 core-agent 增加 metrics env 和内部 `expose: 9108`。
   - `deploy/core-agent/systemd/core-agent.env.example` 增加 metrics 配置项。
   - 文档更新：
     - `docs/observability.md` / `docs/observability.zh-CN.md`
     - `docs/deployment.md` / `docs/deployment.zh-CN.md`
     - `deploy/core-agent/systemd/README.md` / `README.zh-CN.md`
     - `deploy/one-click/ubuntu-25.10/README.md` / `README.zh-CN.md`
   - `progress.md` 的 `P2-2` 已同步为“已有 core-agent 独立 metrics”。

本轮修改文件

- `.claude/change-cache.md`
- `.claude/progress.md`
- `.env.example`
- `core-agent/Cargo.toml`
- `core-agent/src/config/mod.rs`
- `core-agent/src/main.rs`
- `core-agent/src/api/grpc_server.rs`
- `core-agent/src/observability/mod.rs`
- `core-agent/src/observability/metrics.rs`
- `docker-compose.yml`
- `docs/observability.md`
- `docs/observability.zh-CN.md`
- `docs/deployment.md`
- `docs/deployment.zh-CN.md`
- `deploy/core-agent/systemd/core-agent.env.example`
- `deploy/core-agent/systemd/README.md`
- `deploy/core-agent/systemd/README.zh-CN.md`
- `deploy/one-click/ubuntu-25.10/README.md`
- `deploy/one-click/ubuntu-25.10/README.zh-CN.md`

本地验证

- 已通过：
  - `go test ./internal/grpcclient ./internal/middleware ./internal/api`
- 未验证：
  - `cargo fmt`
  - `cargo test`
  - 原因：当前环境无 `cargo`。

commit摘要

- 计划提交：`feat(core-agent): add standalone prometheus metrics endpoint`

希望接下来的 AI 做什么

1. 先在 CI 或具备 Rust 工具链环境验证 `core-agent` 编译与测试，重点关注：
   - `axum` 新增依赖兼容性
   - `grpc_server.rs` 中 `observe_grpc_call` 包装后的 trait 方法签名兼容性
2. 若通过，继续推进 `P2-2` 剩余项：
   - 统一 OTel collector/exporter 方案
   - metrics retention / alert baseline
3. 若失败，优先修正 Rust 编译问题，再推进 OTel 设计。

by: gpt-5.5
