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

当前关键指标族包括：

- `snowpanel_http_requests_total`
- `snowpanel_http_request_duration_seconds`
- `snowpanel_http_requests_in_flight`
- `snowpanel_agent_requests_total`
- `snowpanel_agent_request_duration_seconds`

其中 agent RPC 指标包含以下标签：

- `rpc`
- `outcome`（`success` / `error`）
- `transport`（`true` / `false`）

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

## 当前缺口

- 尚未接入完整 OpenTelemetry 管线。
- 尚未接入分布式追踪后端（Jaeger/Tempo 等）。
- core-agent 暂无独立 Prometheus 指标端点（目前通过 backend 侧 RPC 指标 + agent 日志进行观测）。

