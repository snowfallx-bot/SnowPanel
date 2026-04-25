【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮继续推进 `P2-2`，把“Prometheus + Alertmanager baseline”往前推进到“分布式 tracing baseline”，补出 `backend -> core-agent -> OTel Collector -> Jaeger` 的最小闭环。

本次核心判断

1. 仅有 metrics + request-id 还不够，跨服务延迟/错误排查依旧缺真正的 span 级视图。
2. 最稳妥的下一步不是直接接某个云厂商 tracing backend，而是先在仓库内落地通用 OTLP collector/exporter 策略，再让后续通知/采集后端替换 exporter。
3. backend 侧可以低侵入接官方 `otelgin` + `otelgrpc`；core-agent 侧需要手动把 tonic metadata 中的 trace context 提取出来并挂到 gRPC server span 上。

本轮实际改动

1. backend 接入 OTel tracing
   - 新增 `backend/internal/observability/tracing.go`：
     - 基于 OTLP gRPC exporter 初始化 tracer provider。
     - 支持环境变量：
       - `OTEL_TRACING_ENABLED`
       - `OTEL_SERVICE_NAME`
       - `OTEL_SERVICE_VERSION`
       - `OTEL_EXPORTER_OTLP_ENDPOINT`
       - `OTEL_EXPORTER_OTLP_INSECURE`
       - `OTEL_TRACES_SAMPLER_ARG`
   - `backend/cmd/server/main.go`：
     - 启动时初始化 tracing，失败时仅 warn，不阻塞服务主链路。
     - 退出时调用 tracer shutdown。
   - `backend/internal/api/router.go`：
     - 接入 `otelgin` HTTP middleware。
   - `backend/internal/grpcclient/agent_client.go`：
     - 接入 `otelgrpc.NewClientHandler()`，让 backend -> core-agent 的 gRPC 调用自动生成 client span 并传播 trace context。
   - `backend/internal/middleware/*`：
     - 新增 span request_id 属性注入中间件。
     - access log 追加 `trace_id` / `span_id`。
     - panic recover 会记录 span error/status。
   - `backend/internal/config/config.go`：
     - 增加 tracing 配置解析。

2. core-agent 补 tracing 配置与 gRPC trace context 提取
   - `core-agent/Cargo.toml`：
     - 新增 `opentelemetry` / `opentelemetry_sdk` / `opentelemetry-otlp` / `tracing-opentelemetry` 依赖声明。
   - `core-agent/src/observability/tracing.rs`：
     - 新增 tracing subscriber + OTLP exporter 初始化。
   - `core-agent/src/main.rs`：
     - 启动时优先初始化 OTel tracing，失败时 fallback 到原 fmt logger。
   - `core-agent/src/config/mod.rs`：
     - 增加 `APP_ENV` 与 OTEL 相关环境变量解析。
   - `core-agent/src/api/grpc_server.rs`：
     - 为各 gRPC handler 统一创建 server span。
     - 从 tonic metadata 提取远端 trace context。
     - 将 span 挂到当前 gRPC 请求处理 future 上。
     - 保留原有 request_id 日志链路。

3. observability stack 新增 collector + Jaeger
   - `docker-compose.observability.yml`：
     - 新增 `otel-collector` 与 `jaeger`。
     - 在 observability 覆盖模式下，为 backend / 容器版 core-agent 注入默认 OTLP tracing 环境变量。
   - 新增 `deploy/observability/otel-collector/config.yaml`：
     - `otlp` receiver
     - `batch` processor
     - exporter 到 `jaeger:4317`

4. 环境变量、systemd 模板与文档同步
   - `.env.example`：
     - 新增 Jaeger / OTel Collector 端口与 tracing 环境变量模板。
   - `deploy/core-agent/systemd/core-agent.env.example`：
     - 新增 host-agent tracing 所需 OTEL 变量。
   - 更新文档：
     - `docs/observability.md` / `docs/observability.zh-CN.md`
     - `docs/deployment.md` / `docs/deployment.zh-CN.md`
     - `deploy/core-agent/systemd/README.md` / `README.zh-CN.md`
   - 更新 `.claude/progress.md`：
     - 标注 OTel + Jaeger baseline 已进入仓库，`P2-2` 剩余项转为“真实运行验证 + alert 生产化校准”。

本轮修改文件

- `.claude/change-cache.md`
- `.claude/progress.md`
- `.env.example`
- `Makefile`
- `backend/cmd/server/main.go`
- `backend/go.mod`
- `backend/go.sum`
- `backend/internal/api/router.go`
- `backend/internal/config/config.go`
- `backend/internal/grpcclient/agent_client.go`
- `backend/internal/middleware/access_log.go`
- `backend/internal/middleware/recover.go`
- `backend/internal/middleware/tracing.go`
- `backend/internal/observability/tracing.go`
- `core-agent/Cargo.toml`
- `core-agent/src/api/grpc_server.rs`
- `core-agent/src/config/mod.rs`
- `core-agent/src/main.rs`
- `core-agent/src/observability/mod.rs`
- `core-agent/src/observability/tracing.rs`
- `deploy/core-agent/systemd/README.md`
- `deploy/core-agent/systemd/README.zh-CN.md`
- `deploy/core-agent/systemd/core-agent.env.example`
- `deploy/observability/otel-collector/config.yaml`
- `docker-compose.observability.yml`
- `docs/deployment.md`
- `docs/deployment.zh-CN.md`
- `docs/observability.md`
- `docs/observability.zh-CN.md`

本地验证

- 已通过：
  - `cd backend && go test ./...`
- 未做 / 受环境限制：
  - `cargo check` / `cargo test`（当前环境无 `cargo`）
  - `docker compose ... config/up`（当前环境无 `docker`）
  - Jaeger / OTel Collector / cross-service trace 实际联调

风险与注意事项

1. core-agent tracing 改动是按官方 crate API 文档写入的，但当前机器无法用 `cargo` 编译验证；下一位 agent 应优先在有 Rust toolchain 的环境做 `cargo check`，必要时修正细节 API。
2. host-agent 模式下，只有 backend 容器会自动通过 compose override 注入 tracing env；宿主机 `core-agent` 还需要手工在 `/etc/snowpanel/core-agent.env` 中配置 OTEL 变量。

commit摘要

- 计划提交：`feat(observability): add otel tracing pipeline and jaeger baseline`

希望接下来的 AI 做什么

1. 在具备 `cargo` 的环境优先验证并修正 Rust 侧：
   - `cd core-agent && cargo fmt --check`
   - `cd core-agent && cargo check`
   - `cd core-agent && cargo test`
2. 在具备 Docker 的环境验证 tracing 闭环：
   - `make up-observability`
   - backend 发起一个会走 core-agent 的真实请求
   - 在 Jaeger UI 确认单条 trace 中同时出现 backend HTTP span、backend gRPC client span、core-agent gRPC server span
3. 若 tracing 基线验证通过，继续 `P2-2` 剩余项：
   - 真实通知渠道接入 Alertmanager
   - 告警去重 / 升级策略
   - SLO/SLI 阈值校准

by: gpt-5.5
