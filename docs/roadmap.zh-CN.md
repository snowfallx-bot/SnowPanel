# 路线图

语言: [English](roadmap.md) | **简体中文**

这份路线图描述的是仓库当前真实状态，而不是最早的初始化计划。

## 已完成的基础能力

- backend 与 core-agent 的真实 gRPC 主链路已经覆盖 dashboard、files、services、Docker、cron
- 宿主机 Agent 运行模式已落地，包含 systemd 模板与 Ubuntu 一键安装
- secrets、bootstrap admin、内部端口暴露、cron allowlist 等安全基线已收口
- 基于 RBAC 的认证/会话模型已落地，前端也已具备权限感知
- Docker / service restart 已接入真实异步任务执行
- 文件能力已扩展到更接近真实运维场景
- CI 已形成分层覆盖：backend tests、compose smoke、backend integration、frontend e2e

## P2 已完成项

### P2-2 生产化观测能力

- Prometheus 指标与基线告警规则已具备
- Alertmanager 基线路由已具备
- OTEL tracing 基线现已具备：
  - backend HTTP spans
  - backend gRPC client spans
  - core-agent gRPC server spans
  - OTel Collector -> Jaeger 管线
- Prometheus/Alertmanager 的 SLO 基线已扩展：
  - backend 可用性 recording rules 与 warning/critical 分级告警
  - core-agent gRPC 错误率 recording rule 与 warning/critical 分级告警
  - Alertmanager warning/critical 双接收器路由基线
- observability 冒烟脚本现已具备：
  - Jaeger 跨服务 trace 校验（`scripts/observability/trace-smoke.ps1`）
  - Alertmanager 合成告警注入校验（`scripts/observability/alertmanager-smoke.ps1`）
  - tracing + alertmanager 一键串行校验（`scripts/observability/full-smoke.ps1`）
- observability 配置校验闸门现已具备：
  - `scripts/observability/validate-config.ps1`（`promtool`/`amtool` 检查，默认 Docker 且支持本地回退 + `promtool test rules`）
  - `ci.yml` 中的 `observability-config` 任务
- Alertmanager 生产落地辅助能力已补齐：
  - 生产接收器模板：`deploy/observability/alertmanager/alertmanager.production.example.yml`
  - 生产配置生成脚本：`scripts/observability/generate-alertmanager-config.ps1`
- SLO burn-rate 已扩展为 5m/30m 双窗口，并具备 warning/critical 分级告警
- compose + host-agent 双模式观测冒烟通过证据已沉淀到：
  - `docs/observability-validation.md`
  - `docs/observability-validation.zh-CN.md`

### P2-3 原型痕迹清理

- backend README 中过时的原型遗留描述已移除
- 前端应用壳与 e2e 页面锚点已不再使用 `Linux Panel Prototype` 文案
- root README 已补上 observability 命令与文档入口
- README / roadmap / observability 文档中的历史原型措辞与重复说明已对齐收口

## P3 生产加固

### P3-0 稳定性闸门

`p3-production-hardening` 分支上的本地 P3-0 稳定性闸门已完成：

- `make lint` 通过
- `make test` 通过
- backend、core-agent、frontend 分模块门禁通过
- `make proto-go` 已可在 Windows 本地环境通过 Makefile 自动发现工具链运行
- 已提交的 Go protobuf bindings 与当前本地 proto 工具链一致
- Compose mode 的 `/health` 与 `/ready` 冒烟通过
- Host-agent mode 的 `/health` 与 `/ready` 冒烟通过
- 验证证据与本地环境说明已记录到 `docs/p3-stabilization-report.md`

该 milestone 推送后仍需等待远端 CI 作为最终确认。

### P3-1 告警投递与运维治理

本地 P3-1 告警治理已完成：

- 已新增告警运行手册 `docs/alerting-runbook.zh-CN.md`
- 已文档化 warning/critical 归属、paging 策略、升级路径、去重节奏、inhibition、silence、回滚与合成告警测试流程
- observability 文档已链接告警运行手册
- `scripts/observability/validate-config.ps1` 已支持额外校验生成的 Alertmanager 配置文件
- Prometheus config/rules/tests、Alertmanager baseline、production example、generated production config 校验通过
- warning、critical、inhibition 合成告警冒烟通过
- 本地证据已记录到 `docs/observability-validation.zh-CN.md`

该 milestone 推送后仍需等待远端 CI 作为最终确认。

### P3-2 Backend 到 Core-Agent 信任边界

进行中：

- backend -> core-agent gRPC metadata token auth mode 已实现。
- core-agent 在 token mode 下会拒绝缺失或错误的 `x-snowpanel-agent-token`。
- 本地开发仍保留 `none` 模式。
- `mtls` 配置项已预留，并在证书支持实现前 fail fast。
- backend 会将 gRPC `Unauthenticated` 映射为 `core agent authentication failed`。
- 测试已覆盖 token metadata 注入、auth failure 映射、core-agent allow/deny 路径，并确认错误不泄露 token 值。
- 本地 token-auth compose smoke 已通过 `/health` 与 `/ready`。

### P3-3 Secrets & Settings Hardening

本地 P3-3 secrets and settings hardening 已完成：

- audit request summary、audit result message、task log metadata 在持久化前会对常见敏感字段做脱敏。
- 前端 token storage 决策已记录在 `docs/security-token-storage-decision.md`。
- AES-256-GCM 加密 helper 与 `SNOWPANEL_ENCRYPTION_KEY` 配置已实现。
- SystemSetting repository/service encryption 已支持 `is_encrypted=true` 值。
- startup validation 会拒绝无效 encryption key，并在已有 encrypted settings 但缺少 key 时 fail fast。
- 本地门禁已通过：`go test ./...`、`make lint`、`make test`。

该 milestone 推送后仍需等待远端 CI 作为最终确认。

### P3-4 Durable Task Worker

本地 P3-4 durable worker 接线正在推进：

- durable task schema 字段已通过 `backend/migrations/0002_durable_tasks.*.sql` 新增。
- task model 已包含 lease、retry、idempotency、next-run 元数据。
- worker 配置环境变量已加入 `TASK_WORKER_*`。
- TaskRepository 已暴露 claim、heartbeat、complete、fail、stale-release API。
- backend 启动时已在 `TASK_WORKER_ENABLED=true` 下运行 DB-backed worker。
- 任务创建默认只入队，由 durable worker claim 后执行；关闭 worker 时仍保留 legacy goroutine executor 作为兼容/回滚路径。
- Docker/service restart 任务创建已支持可选 `idempotency_key`；重复 key 会返回已有任务，不会重复入队。
- worker loop 已支持 concurrency、lease heartbeat、stale lease release、max-attempt retry、retry backoff 与基于 context 的 graceful shutdown。
- retry policy 已区分 retryable failure 与 non-retryable validation/payload failure。
- task/worker metrics 已暴露 queue depth、running count、completion totals、task duration 与 claim outcomes。
- service tests 已覆盖 enqueue-only、idempotency duplicate、worker claim/execute、worker context cancel、claim-only-once、stale lease recovery、失败重试、non-retryable validation failure、max-attempt 耗尽、排队/运行中取消、长任务 heartbeat 与 task metric 记录。
- 本地 backend 门禁已通过：`cd backend && go test ./...`。

### P3-5 Host-Agent Least Privilege

本地 P3-5 宿主机 agent 最小权限加固正在推进：

- core-agent production 配置已默认拒绝不安全配置，除非显式设置 `CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=true`。
- production deny-by-default 校验覆盖未启用认证、allowed roots 包含 `/`、服务白名单为空、Cron 命令白名单依赖默认值。
- 宿主机操作类别可通过 `CORE_AGENT_ENABLE_FILE_OPS`、`CORE_AGENT_ENABLE_SERVICE_OPS`、`CORE_AGENT_ENABLE_DOCKER_OPS`、`CORE_AGENT_ENABLE_CRON_OPS` 分别关闭。
- gRPC 操作入口会在执行文件、服务、Docker、Cron 动作前检查 feature gate。
- core-agent 启动会对过宽 roots、空 service whitelist、Cron 默认 allowlist、非 loopback metrics、unsafe override、未认证 wildcard gRPC 绑定输出警告。
- systemd 部署模板已增加基础 sandboxing，并记录 Docker/systemd 兼容性取舍。
- 部署与安全文档已加入宿主机 agent 生产检查表。

### P3-6 Audit Retention, Export, and Forensics

本地 P3-6 审计取证能力正在推进：

- audit 记录已包含 `request_id` 与 `trace_id`，便于从 backend log 与 trace 回查审计记录。
- audit log filter 已覆盖时间范围、user id、username、module、action、target、success、result code、request id、trace id。
- 审计导出接口已新增：`GET /api/v1/audit/logs/export?format=csv|jsonl`。
- `audit.export` 权限已加入种子数据，并默认分配给 `super_admin`。
- 导出按分页读取审计记录，最多导出 100000 行，并复用已脱敏的 audit 字段。
- `AUDIT_RETENTION_DAYS` 与 `AUDIT_EXPORT_MAX_ROWS` 已用于配置清理 cutoff 与导出规模。
- audit retention cleanup 已通过 `POST /api/v1/audit/retention/cleanup` 暴露，要求 `audit.manage`，并会审计 cleanup 尝试本身。
- audit UI 已提供取证筛选、CSV/JSONL 导出、日志详情面板、request_id copy，以及 task 审计记录到任务详情的链接。
- tasks UI 已支持通过 `/tasks?task_id=<id>` 直接打开任务详情。
- backend tests 已覆盖 filter 传递、CSV export、JSONL export、request/trace id 持久化、retention dry-run 与实际 delete 行为。

## 后续加固（Post-P3-0）

1. 按团队值班制度把最终告警目的地接入到真实 on-call 通道
2. 基于真实流量持续调优去重/升级窗口与 SLO 阈值
3. 若后续排障深度需要，再补浏览器侧 tracing
4. 实现 backend 与 core-agent 之间的认证边界
5. 增加敏感配置静态加密与生产启动校验
6. 将当前 goroutine 异步任务升级为 DB-backed durable worker

## 当前不优先的方向

- UI 美化或视觉重做
- 在运维主链路未补齐前继续扩页面
- 在运维治理未固化前过早把项目表述成“全面生产就绪”
