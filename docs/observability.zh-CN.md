# 可观测性说明

语言: [English](observability.md) | **简体中文**

## 范围

本文描述 SnowPanel 当前可用于生产排障的基础能力：

- 指标（Prometheus 格式）
- 跨服务请求关联（`X-Request-ID`）
- 分布式追踪（OTLP -> OTel Collector -> Jaeger）
- 日志检索路径

## 指标

backend 暴露 Prometheus 指标端点：

- `GET /metrics`

core-agent 在启用时也会暴露独立 Prometheus 端点：

- `GET http://<CORE_AGENT_METRICS_HOST>:<CORE_AGENT_METRICS_PORT>/metrics`

当前关键指标族包括：

- `snowpanel_http_requests_total`
- `snowpanel_http_request_duration_seconds`
- `snowpanel_http_requests_in_flight`
- `snowpanel_agent_requests_total`
- `snowpanel_agent_request_duration_seconds`
- `snowpanel_core_agent_grpc_requests_total`
- `snowpanel_core_agent_grpc_request_duration_seconds`
- `snowpanel_core_agent_grpc_requests_in_flight`

Prometheus 还提供了面向 SLO 的 recording rules：

- `snowpanel:backend_http_total:rate5m`
- `snowpanel:backend_http_5xx:rate5m`
- `snowpanel:backend_http_availability:ratio5m`
- `snowpanel:core_agent_grpc_error_ratio:ratio5m`

其中 agent RPC 指标包含以下标签：

- `rpc`
- `outcome`（`success` / `error`）
- `transport`（`true` / `false`）

其中 core-agent gRPC 指标包含以下标签：

- `grpc_method`
- `outcome`（`ok` / `error`）

## 可观测性基线栈

仓库已提供可直接落地的可观测性基线部署：

- Compose 覆盖文件：`docker-compose.observability.yml`
- 抓取配置：`deploy/observability/prometheus/prometheus.yml`
- 告警规则：`deploy/observability/prometheus/alerts/snowpanel-alerts.yml`
- Alertmanager 路由配置：`deploy/observability/alertmanager/alertmanager.yml`
- OTel Collector 配置：`deploy/observability/otel-collector/config.yaml`
- Jaeger UI：`http://127.0.0.1:${JAEGER_UI_PORT:-16686}`

启动方式：

- Compose 模式：`make up-observability`
- 宿主机 Agent 模式：`make up-host-agent-observability`

查看可观测性入口：

- `http://127.0.0.1:${PROMETHEUS_PORT:-9090}`
- `http://127.0.0.1:${ALERTMANAGER_PORT:-9093}`
- `http://127.0.0.1:${JAEGER_UI_PORT:-16686}`

停止方式：

- Compose 模式：`make down-observability`
- 宿主机 Agent 模式：`make down-host-agent-observability`

说明：

- 基线抓取目标默认假设 backend `:8080` 与 core-agent metrics `:9108`。
- 若你的运行端口不同，请同步修改 `deploy/observability/prometheus/prometheus.yml`。
- Alertmanager 默认 warning/critical 接收器均为 no-op；请在 `deploy/observability/alertmanager/alertmanager.yml` 中配置 webhook/邮件/IM 等真实通知通道。
- Compose 可观测性模式会默认给 `backend` 与容器版 `core-agent` 打开 OTLP tracing 导出。
- 若使用宿主机 Agent 模式，还需要在 `deploy/core-agent/systemd/core-agent.env.example`（或 `/etc/snowpanel/core-agent.env`）里设置 OTEL 环境变量，让宿主机上的 `core-agent` 把 trace 发往 collector。

## 请求链路关联

当前已支持 request-id 从 backend 透传到 core-agent：

1. backend HTTP 中间件接收 `X-Request-ID`（若缺失则自动生成）。
2. request-id 写入请求上下文，并回写到响应头。
3. backend gRPC client 将其作为 `x-request-id` metadata 发送给 core-agent。
4. core-agent 对每个 gRPC 调用记录：
   - `request_id`
   - `grpc_method`

这样可以把同一次请求从浏览器/API 客户端日志一路关联到 backend 与 core-agent。

## 分布式追踪基线

当前 trace 链路如下：

1. backend HTTP 请求创建 server span。
2. backend gRPC client 创建子 span，并将 W3C trace context 透传给 core-agent。
3. core-agent 提取上游 context，创建同一条 trace 下的 gRPC server span。
4. 两端统一通过 OTLP 导出到 OTel Collector。
5. Collector 做 batch 后转发到 Jaeger。

推荐 OTEL 环境变量：

- `OTEL_TRACING_ENABLED=true`
- `OTEL_EXPORTER_OTLP_ENDPOINT=<collector-host>:4317`
- `OTEL_EXPORTER_OTLP_INSECURE=true`
- `OTEL_TRACES_SAMPLER_ARG=1.0`

默认服务名：

- backend：`snowpanel-backend`
- core-agent：`snowpanel-core-agent`

## Tracing 实测清单

可按以下清单在 compose 或 host-agent 模式验证 trace 链路是否打通：

1. 启动可观测性栈：
   - Compose 模式：`make up-observability`
   - 宿主机 Agent 模式：`make up-host-agent-observability`
2. 先登录拿到可用 bearer token（访问 `/api/v1/*` 受保护接口需要）。
3. 调用一个必经 core-agent 的 backend 接口，例如：
   - `GET /api/v1/dashboard/summary`
   - 并携带自定义请求头 `X-Request-ID: trace-e2e-001`
4. 确认 backend 响应头回写了同一个 `X-Request-ID`。
5. 在 Jaeger（`http://127.0.0.1:${JAEGER_UI_PORT:-16686}`）确认同一条 trace 里同时出现以下服务的 span：
   - `snowpanel-backend`
   - `snowpanel-core-agent`
6. 若是宿主机 Agent 模式，再确认 `/etc/snowpanel/core-agent.env`（或 `deploy/core-agent/systemd/core-agent.env.example`）中的 OTEL 变量配置正确。

可选辅助脚本（PowerShell）：

```powershell
pwsh -File ./scripts/observability/trace-smoke.ps1 `
  -AccessToken "<access_token>" `
  -BackendBaseUrl "http://127.0.0.1:8080" `
  -JaegerBaseUrl "http://127.0.0.1:16686"
```

该脚本会带自动生成的 `X-Request-ID` 调用 `GET /api/v1/dashboard/summary`，随后轮询 Jaeger；若未找到同时包含 `snowpanel-backend` 与 `snowpanel-core-agent` 的近期 trace，会直接失败。

另见：[`scripts/observability/README.md`](../scripts/observability/README.md)，集中说明 tracing 与 Alertmanager 冒烟脚本用法。
若希望一次跑完两项校验，可使用：`pwsh -File ./scripts/observability/full-smoke.ps1 -AccessToken "<access_token>"`。
`full-smoke.ps1` 也支持 `-LoginUsername` + `-LoginPassword` 自动登录模式，无需手工提取 token。

## 快速排障路径

1. 从浏览器开发者工具或 API 响应头拿到 `X-Request-ID`。
2. 用该 `request_id` 检索 backend 日志。
3. 用相同 `request_id` 检索 core-agent 日志。
4. 同时查看 `/metrics` 中是否出现上升：
   - `snowpanel_http_request_duration_seconds`
   - `snowpanel_agent_request_duration_seconds`
   - `snowpanel_agent_requests_total{outcome="error",...}`
5. 查看 core-agent 指标是否出现方法级热点：
   - `snowpanel_core_agent_grpc_requests_total`
   - `snowpanel_core_agent_grpc_request_duration_seconds`
   - `snowpanel_core_agent_grpc_requests_in_flight`

## 基线告警项

默认告警包括：

- `SnowPanelBackendDown`
- `SnowPanelCoreAgentMetricsDown`
- `SnowPanelBackendP95LatencyHigh`
- `SnowPanelBackendP95LatencyCritical`
- `SnowPanelCoreAgentP95LatencyHigh`
- `SnowPanelCoreAgentP95LatencyCritical`
- `SnowPanelBackendAgentTransportErrorsHigh`
- `SnowPanelCoreAgentGrpcErrorRateHigh`
- `SnowPanelCoreAgentGrpcErrorRateCritical`
- `SnowPanelCoreAgentInFlightHigh`
- `SnowPanelBackendAvailabilitySLOWarning`
- `SnowPanelBackendAvailabilitySLOCritical`

## 告警投递基线

Prometheus 默认会把告警发送到 Alertmanager（`alertmanager:9093`）。

默认路由策略：

- `severity="warning"` -> `snowpanel-warning`
- `severity="critical"` -> `snowpanel-critical`

其中 `snowpanel-warning` 与 `snowpanel-critical` 默认都给出注释模板（webhook 示例），便于按团队规范接入真实通知渠道。

## Alertmanager 落地清单

从基线 no-op 路由切到真实生产告警投递时，可按以下清单执行：

1. 在 `deploy/observability/alertmanager/alertmanager.yml` 中将 `snowpanel-warning` 与 `snowpanel-critical` 两个接收器都替换为真实通道（webhook/邮件/slack/wechat 等）。
2. 按严重级别明确路由归属：
   - `critical` -> 值班/分页通道
   - `warning` -> 非分页运维通道（必要时新增独立 receiver/route）
3. 保留或扩展抑制规则，让同一 `alertname` 下 `critical` 自动抑制重复 `warning` 噪音。
4. 发布并校验路由配置：
   - `make up-observability`（或 `make up-host-agent-observability`）
   - 在 Alertmanager UI（`/#/status`）确认 receiver 与路由状态
5. 做一次可控投递验证：
   - 可直接注入一条合成告警：
     - `pwsh -File ./scripts/observability/alertmanager-smoke.ps1`
   - 或临时下调 `deploy/observability/prometheus/alerts/snowpanel-alerts.yml` 中某条告警阈值
   - 再按需生成匹配负载/流量
   - 确认通知在去重窗口内只发送一次，且包含 `alertname`、`severity`、`instance` 等关键标签
6. 验证后恢复原阈值，并提交最终配置/规则。

## 当前缺口

- 前端/浏览器侧 tracing 尚未接入。
- 尚未形成基于 trace 的日志统一采集链路；当前仍主要依赖日志检索 + `X-Request-ID` 关联。
- 告警通知、去重、升级策略与 SLO/SLA 阈值仍需按真实生产负载校准。
