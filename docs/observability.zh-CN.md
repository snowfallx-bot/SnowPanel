# 可观测性说明

语言: [English](observability.md) | **简体中文**

## 范围

本文描述 SnowPanel 当前可用于生产排障的基础能力：

- 指标（Prometheus 格式）
- 跨服务请求关联（`X-Request-ID`）
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

其中 agent RPC 指标包含以下标签：

- `rpc`
- `outcome`（`success` / `error`）
- `transport`（`true` / `false`）

其中 core-agent gRPC 指标包含以下标签：

- `grpc_method`
- `outcome`（`ok` / `error`）

## Prometheus 基线栈

仓库已提供可直接落地的 Prometheus 基线部署：

- Compose 覆盖文件：`docker-compose.observability.yml`
- 抓取配置：`deploy/observability/prometheus/prometheus.yml`
- 告警规则：`deploy/observability/prometheus/alerts/snowpanel-alerts.yml`
- Alertmanager 路由配置：`deploy/observability/alertmanager/alertmanager.yml`

启动方式：

- Compose 模式：`make up-observability`
- 宿主机 Agent 模式：`make up-host-agent-observability`

查看 Prometheus：

- `http://127.0.0.1:${PROMETHEUS_PORT:-9090}`
- `http://127.0.0.1:${ALERTMANAGER_PORT:-9093}`

停止方式：

- Compose 模式：`make down-observability`
- 宿主机 Agent 模式：`make down-host-agent-observability`

说明：

- 基线抓取目标默认假设 backend `:8080` 与 core-agent metrics `:9108`。
- 若你的运行端口不同，请同步修改 `deploy/observability/prometheus/prometheus.yml`。
- Alertmanager 默认接收器为 no-op；请在 `deploy/observability/alertmanager/alertmanager.yml` 中配置 webhook/邮件/IM 等真实通知通道。

## 请求链路关联

当前已支持 request-id 从 backend 透传到 core-agent：

1. backend HTTP 中间件接收 `X-Request-ID`（若缺失则自动生成）。
2. request-id 写入请求上下文，并回写到响应头。
3. backend gRPC client 将其作为 `x-request-id` metadata 发送给 core-agent。
4. core-agent 对每个 gRPC 调用记录：
   - `request_id`
   - `grpc_method`

这样可以把同一次请求从浏览器/API 客户端日志一路关联到 backend 与 core-agent。

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
- `SnowPanelCoreAgentP95LatencyHigh`
- `SnowPanelBackendAgentTransportErrorsHigh`
- `SnowPanelCoreAgentGrpcErrorRateHigh`
- `SnowPanelCoreAgentInFlightHigh`

## 告警投递基线

Prometheus 默认会把告警发送到 Alertmanager（`alertmanager:9093`）。

默认路由策略：

- 所有告警 -> `snowpanel-null`
- `severity="critical"` -> `snowpanel-critical`

其中 `snowpanel-critical` 默认给出注释模板（webhook 示例），便于按团队规范接入真实通知渠道。

## 当前缺口

- 尚未接入完整 OpenTelemetry 管线。
- 尚未接入分布式追踪后端（Jaeger/Tempo 等）。
- backend 与 core-agent 之间尚未形成统一 OTel collector/exporter 策略。
