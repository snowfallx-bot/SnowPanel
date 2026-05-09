# 可观测性

语言: [English](observability.md) | **简体中文**

SnowPanel 当前已经提供 backend metrics、结构化请求日志、健康检查、就绪检查、审计日志，以及 core-agent tracing 日志。

## HTTP 健康检查

- `GET /health` 用于判断 backend 进程是否存活。
- `GET /ready` 用于判断依赖是否就绪，包括数据库与 core-agent 连接。
- `GET /metrics` 暴露 backend 的 Prometheus 指标。

## Backend Metrics

backend 使用 `snowpanel` namespace 暴露 Prometheus 指标：

- `snowpanel_http_requests_total`
  - labels: `method`, `route`, `status`
  - 统计 Gin 处理的 HTTP 请求数
- `snowpanel_http_request_duration_seconds`
  - labels: `method`, `route`
  - 记录 HTTP 请求耗时
- `snowpanel_http_requests_in_flight`
  - 记录当前进行中的 HTTP 请求数
- `snowpanel_agent_requests_total`
  - labels: `outcome`, `transport`
  - 统计 backend 到 core-agent 的 gRPC 调用数
- `snowpanel_agent_request_duration_seconds`
  - labels: `outcome`, `transport`
  - 记录 core-agent gRPC 调用耗时

Prometheus scrape 示例：

```yaml
scrape_configs:
  - job_name: snowpanel-backend
    static_configs:
      - targets:
          - backend:8080
```

## 日志与关联

- backend 每个请求都会通过 request-id middleware 获得 request id。
- access log 包含 `request_id`、method、path、status、latency、client IP、user agent。
- core-agent 使用 `tracing` 日志；生产环境应从宿主机上的 host-agent service 日志采集。
- audit logs 记录面向用户的管理操作，并可通过审计 API 和 UI 检索。

## 生产排障检查顺序

1. 先看 backend `/ready`，区分进程存活和依赖就绪问题。
2. 查看 `snowpanel_agent_requests_total{outcome="error"}` 以及 `transport` label，判断 backend 到 agent 是否异常。
3. 用 backend access log 中的 `request_id` 关联单次请求。
4. 当 agent transport error 增加时，检查宿主机 core-agent service 日志。
5. 用 audit logs 确认状态变更操作的发起人和动作。

## 当前范围

当前里程碑已经覆盖 backend HTTP 与 backend 到 core-agent 调用的 Prometheus metrics。完整分布式 tracing 不是当前发布阻塞项；后续若引入，应基于现有 request id 和结构化日志继续扩展。
