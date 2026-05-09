# 告警运行手册

语言: [English](alerting-runbook.md) | **简体中文**

这份手册定义 SnowPanel 告警的归属、分流、首轮排障和回滚路径。

## 归属

| 严重级别 | 负责人 | 策略 |
| --- | --- | --- |
| `warning` | SnowPanel 平台负责人 | 非 paging。工作时间或下一个维护窗口处理。 |
| `critical` | SnowPanel 值班负责人 | paging。需要立即确认；若 15 分钟内无法缓解或确认有服务影响，需要升级。 |
| `critical` escalation | SnowPanel incident lead | paging 升级通道。用于主 critical 通道未确认或未缓解的场景。 |

receiver 映射：

- `snowpanel-warning`：warning 通道
- `snowpanel-critical`：主 critical paging 通道
- `snowpanel-critical-escalation`：critical 二级升级通道

## 路由与去重

- 分组标签：`alertname`、`severity`、`instance`
- Warning 节奏：`group_wait=30s`、`group_interval=10m`、`repeat_interval=4h`
- Critical 节奏：`group_wait=10s`、`group_interval=2m`、`repeat_interval=30m`
- Critical 升级节奏：`group_wait=2m`、`group_interval=10m`、`repeat_interval=15m`
- 抑制规则：当 `alertname` 和 `instance` 相同，`critical` 告警会抑制对应 `warning` 告警。

生产配置建议通过脚本生成：

```powershell
pwsh -File ./scripts/observability/generate-alertmanager-config.ps1 `
  -WarningWebhookUrl "https://alerts.example.com/snowpanel-warning" `
  -CriticalWebhookUrl "https://alerts.example.com/snowpanel-critical" `
  -CriticalEscalationWebhookUrl "https://alerts.example.com/snowpanel-critical-escalation" `
  -OutputPath "deploy/observability/alertmanager/alertmanager.generated.yml"
```

上线前必须校验：

```powershell
pwsh -File ./scripts/observability/validate-config.ps1
```

## 合成告警测试

在运行中的 Alertmanager 上分别验证 warning 和 critical 路由：

```powershell
pwsh -File ./scripts/observability/alertmanager-smoke.ps1 `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093" `
  -Severity warning

pwsh -File ./scripts/observability/alertmanager-smoke.ps1 `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093" `
  -Severity critical
```

验证 inhibition：

```powershell
pwsh -File ./scripts/observability/alertmanager-inhibition-smoke.ps1 `
  -AlertmanagerBaseUrl "http://127.0.0.1:9093"
```

## 排障流程

### `SnowPanelBackendDown`

1. 检查 backend 容器或服务状态。
2. 检查 `GET /health` 和 `GET /ready`。
3. 查看 backend 日志中的启动错误、数据库连接错误和 panic recovery 记录。
4. 确认 Prometheus 能访问 backend `/metrics`。
5. 如果 backend 在发布后 crash-loop，回滚 backend 镜像或配置变更。

### `SnowPanelCoreAgentMetricsDown`

1. 先确认当前是 compose-agent 还是 host-agent 模式。
2. Compose mode：检查 `snowpanel-core-agent` 容器状态和日志。
3. Host-agent mode：检查宿主机 `core-agent` systemd 服务和 `CORE_AGENT_METRICS_*` 监听配置。
4. 确认防火墙允许 Prometheus 访问 metrics endpoint。
5. 如果 metrics 不可用但 `/ready` 返回 `agent=up`，按观测降级处理；如果 `/ready` 返回 `agent=down`，按运维能力影响处理。

### backend p95 latency high

1. 判断延迟是否集中在某个 route，还是影响所有 backend 流量。
2. 对比 backend CPU、内存与数据库健康状态。
3. 查看 agent RPC latency 指标，判断 backend 是否在等待 core-agent。
4. 在 Jaeger 中查看近期慢 trace，定位耗时 span。
5. 如果问题跟随发布出现，回滚 backend 或 agent 变更。

### core-agent gRPC error ratio high

1. 按 `rpc`、`outcome`、`transport` 检查 backend agent RPC 指标。
2. 使用 backend 日志中的同一个 `request_id` 查询 core-agent 日志。
3. 确认 safe-root、service whitelist、Docker、cron allowlist 错误是预期拒绝，而不是基础设施故障。
4. 如果 transport errors 升高，检查宿主机可达性、防火墙和 core-agent 进程状态。
5. 如果错误与近期发布相关，回滚 agent 配置或二进制。

## 关联查询

使用 `X-Request-ID` 将用户可见错误与 backend/core-agent 日志关联：

1. 从 HTTP 响应头或前端错误详情中获取 `X-Request-ID`。
2. 用该 request ID 搜索 backend 日志。
3. 用同一个 request ID 搜索 core-agent 日志。
4. 在 Jaeger 中搜索近期 trace：`snowpanel.request_id=<request_id>`。
5. 对 agent-backed 操作，确认 trace 中同时包含 `snowpanel-backend` 与 `snowpanel-core-agent` span。

## 临时 Silence

只有在有人明确处理问题或计划维护窗口内，才允许 silence。

建议匹配标签：

- `alertname`
- `instance`
- `severity`

silence 时间应尽量短：

- Warning 维护：最长 4 小时
- Critical 缓解：30 到 60 分钟，之后重新评估

silence comment 中必须写明原因、负责人、回滚方式或后续工单。

## 回滚

1. 将生成的 Alertmanager 配置回滚到上一版已知可用文件。
2. 执行 `pwsh -File ./scripts/observability/validate-config.ps1`。
3. 按部署方式重启或 reload Alertmanager。
4. 重新执行 warning 和 critical 合成告警冒烟。
5. 如适用，确认 resolved 通知也能送达。

本地开发可以继续使用 no-op baseline receiver。生产环境必须接入真实 warning 与 critical receiver 后，才能宣称告警投递就绪。
