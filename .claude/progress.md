# SnowPanel Progress 与长期执行计划

## 当前结论

当前项目不应优先进入 UI 美化或大规模新页面开发。下一阶段应进入：

```text
P3-0 Stabilization Gate
P3-1 Production Hardening & Operational Governance
```

核心目标：在继续扩展 Host、Website、Database、Backup、Plugin 等模块前，先把现有真实运维能力做成可测试、可部署、可审计、可告警、可恢复、可安全控制的生产级基础。

---

## Progress 状态

### 已完成：P2

```text
P2 Completed
- backend <-> core-agent gRPC 主链路已贯通
- dashboard/files/services/docker/cron 已走真实 agent 操作链路
- JWT + RBAC + session validation 已建立
- permission-aware frontend gating 已建立
- Docker/service restart 已有 async task baseline
- file operations 已扩展到实际运维工作流
- observability baseline 已建立：metrics、alerts、tracing、Jaeger、Prometheus、Alertmanager
- CI baseline 已覆盖 backend、core-agent、frontend、proto、compose smoke、integration、e2e
- prototype wording cleanup 已完成
```

### 下一阶段：P3

```text
P3 Production Hardening
- 先跑通并固化稳定性闸门
- 再处理生产告警治理
- 再强化 backend <-> core-agent 信任边界
- 再处理 secrets、task durability、least privilege、audit、backup/restore
```

### P3 当前进展

```text
P3-0 Stabilization Gate
- local quality gates 已通过：make lint、make test、backend/core-agent/frontend 分模块检查
- proto generated files 已通过 make proto-go 刷新并纳入提交
- compose mode /health 与 /ready 冒烟已通过
- host-agent mode /health 与 /ready 冒烟已通过
- docs/p3-stabilization-report.md 已新增验证证据
- docs/roadmap.md 与 docs/roadmap.zh-CN.md 已同步 P3-0 状态
- 远端 CI 仍需在 push 后作为最终确认

P3-1 Alert Delivery & Operational Governance
- alerting runbook 已新增：docs/alerting-runbook.md / docs/alerting-runbook.zh-CN.md
- runbook 已覆盖 warning/critical owner、paging policy、escalation、dedup、inhibition、rollback、synthetic alert procedure
- docs/observability.md / docs/observability.zh-CN.md 已链接 alerting runbook
- validate-config.ps1 已支持额外校验生成的 Alertmanager 配置
- production config 生成脚本已用示例 webhook 成功生成临时配置
- validate-config 已通过：Prometheus config/rules/tests、Alertmanager baseline、production example、generated production config
- synthetic alert smoke 已通过：warning receiver、critical receiver、inhibition
- warning-only 告警规则 fixture 已修正，避免误触发 burn-rate critical
- docs/observability-validation.md / docs/observability-validation.zh-CN.md 已记录本地 P3-1 验证证据
- 远端 CI 仍需在 push 后作为最终确认

P3-2 Backend ↔ Core-Agent Trust Boundary
- token mode 已实现：backend 自动附加 x-snowpanel-agent-token，core-agent 校验 metadata token
- none mode 保持本地开发行为
- mtls mode 已保留配置入口并 fail fast，等待后续证书实现
- backend / core-agent 已增加 token auth 配置校验
- 测试已覆盖 backend token metadata 注入、auth failure 映射、core-agent missing/wrong/correct token、错误不泄露 token
- token auth compose smoke 已通过：/health 与 /ready 均返回 up/ready
- deployment/security/systemd/env 文档已同步 token 模式与生产注意事项

P3-3 Secrets & Settings Hardening
- secret redaction 已新增：audit request summary、audit result message、task log metadata 持久化前脱敏
- redaction 测试已覆盖嵌套 JSON、文本 assignment、audit record、task metadata
- docs/security-token-storage-decision.md / .zh-CN.md 已新增，记录前端 token storage 阶段性决策与迁移前置条件
- AES-256-GCM encryption helper 已新增：支持 32-byte key、versioned payload、wrong-key failure、明文不落 payload
- SNOWPANEL_ENCRYPTION_KEY / SNOWPANEL_ENCRYPTION_KEY_ID 配置入口已新增
- SystemSetting repository/service encryption 已新增：is_encrypted=true 写入前加密，读取时解密
- 测试已覆盖 encrypted write/read、missing key fail、wrong key fail、plaintext setting
- startup validation 已新增：无效 encryption key 拒绝启动；已有 encrypted settings 但缺 key 时 fail fast
- local quality gates 已通过：go test ./...、make lint、make test
- 远端 CI 仍需在 push 后作为最终确认

P3-4 Durable Task Worker
- durable task schema migration 已新增：locked_by、locked_until、attempt、max_attempts、idempotency_key、next_run_at
- task model 已新增 lease/retry/idempotency 字段
- task worker 配置入口已新增：TASK_WORKER_ENABLED/ID/CONCURRENCY/LEASE_DURATION/POLL_INTERVAL/MAX_ATTEMPTS
- TaskRepository 已新增 claim/heartbeat/complete/fail/release stale API 基线
- backend startup 已接入 durable worker：TASK_WORKER_ENABLED=true 时任务只入队，由 DB-backed worker claim 后执行
- TASK_WORKER_ENABLED=false 时保留 legacy goroutine execution，便于本地/回滚兼容
- worker loop 已支持 concurrency、lease heartbeat、stale lease release、max_attempt retry、backoff 与 context-based graceful shutdown
- retry policy 已区分 retryable 与 non-retryable：validation/payload 错误直接失败，普通 agent/transport 错误继续按 backoff 重试
- create task API 已接入 optional idempotency_key，重复 key 返回已有任务而不重复入队
- task/worker metrics 已新增：queue_depth、running、completed_total、duration_seconds、worker claims
- service tests 已覆盖 enqueue-only、idempotency duplicate、worker claim/execute、worker context cancel、claim only once、stale lease recovery、failure retry、non-retryable validation failure、max attempts、queued/running cancellation、long-running heartbeat
- metrics tests 已覆盖 worker claims/completed 打点与 /metrics 暴露
- local backend gate 已通过：cd backend && go test ./...
- 远端 CI 仍需在 push 后作为最终确认

P3-5 Host-Agent Least Privilege
- core-agent production deny-by-default 校验已新增：生产环境拒绝 none auth、/ allowed root、空 service whitelist、未显式配置的 cron allowlist
- CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG 已新增，仅显式开启时跳过 production deny-by-default 检查
- CORE_AGENT_ENABLE_FILE_OPS / SERVICE_OPS / DOCKER_OPS / CRON_OPS 已新增，并在 gRPC 操作入口执行 category gate
- startup warnings 已新增：unsafe override、过宽 allowed roots、空 service whitelist、默认 cron allowlist、非 loopback metrics、未认证 wildcard gRPC
- systemd unit 已补充 ProtectSystem、ProtectHome、ReadWritePaths、RestrictAddressFamilies 等基础 hardening
- deployment/security/systemd docs 已补充生产检查表、feature gate、Docker socket 风险、metrics 暴露、备份前置提醒与 systemd hardening tradeoff
- local core-agent gate 已通过：cargo fmt --all -- --check、cargo test
- local full gate 已通过：make lint、make test

P3-6 Audit Retention, Export, and Forensics
- audit_logs 已新增 request_id / trace_id 字段与索引，用于从 HTTP log / trace 回查审计记录
- audit list filter 已扩展：time range、user_id、username、module、action、target_type、target_id、success、result_code、request_id、trace_id
- audit export endpoint 已新增：GET /api/v1/audit/logs/export?format=csv|jsonl
- audit.export 权限已新增，并默认分配给 super_admin
- audit export 已按分页读取，最多导出 100000 行，导出内容使用已脱敏的 audit 字段
- AUDIT_RETENTION_DAYS / AUDIT_EXPORT_MAX_ROWS 配置已新增，默认分别为 180 / 100000
- audit retention cleanup endpoint 已新增：POST /api/v1/audit/retention/cleanup，要求 audit.manage，并记录 cleanup 审计
- audit UI 已扩展取证筛选、CSV/JSONL 导出、日志详情面板、request_id copy 与 task detail 链接
- tasks UI 已支持 /tasks?task_id=<id> 直接打开任务详情
- service tests 已覆盖 filter 传递、CSV export、JSONL export 与 request/trace id 记录
- service tests 已覆盖 retention dry run 与实际 delete 路径
- local backend gate 已通过：cd backend && go test ./...
- local frontend gate 已通过：npm --prefix frontend run test、npm --prefix frontend run build
- local full gate 已通过：make lint、make test

P3-7 Backup and Restore Foundation
- backup scope 已明确限制为 Postgres metadata、SnowPanel app metadata、observability config snapshot、core-agent config templates 与 backup metadata
- P3 backup 明确排除 raw secret export、任意 filesystem backup、Docker volume backup、remote object storage 与完整 host bare-metal recovery
- docs/restore-drill.md / docs/restore-drill.zh-CN.md 已新增，覆盖 fresh machine restore、Postgres restore、secret rotation、health/ready、login、audit、backup verification 与 rollback
- BackupRepository / BackupService 已新增 metadata create、list、verify 基线
- backup metadata verify 已校验 sha256 checksum 与 size mismatch，并在不匹配时标记 failed
- backup.read / backup.manage 权限已新增，并默认分配给 super_admin
- backup API 已新增：GET /api/v1/backups、POST /api/v1/backups、POST /api/v1/backups/:id/verify
- backup create / verify 操作已记录 audit
- TaskTypeBackupCreate / TaskTypeBackupVerify 已接入 durable task worker baseline
- backup task API 已新增：POST /api/v1/backups/tasks/create、POST /api/v1/backups/:id/verify-task
- backup create task 会创建 metadata 并通过 worker 标记 running/success；backup verify task 会通过 worker 执行 checksum/size verification
- service tests 已覆盖 metadata creation、scope deny、verify success、checksum mismatch failed、list filter normalization
- task service tests 已覆盖 backup create task 与 backup verify task worker execution
- handler tests 已覆盖 list filter 传递、create audit、verify failure audit
- local backend gate 已通过：cd backend && go test ./...
- local full gate 已通过：make lint、make test
```

### 明确暂不优先

```text
Not Current Priority
- UI polish / visual redesign
- 新页面优先开发
- Website/Database/Plugin/Backup 全量功能优先开发
- 过早宣称 fully production-ready
```

---

# Agent 执行规则

## Branch

```bash
git checkout -b p3-production-hardening
```

## Global Rules

1. 不要在 P3 hardening gates 通过前添加大规模新产品面。
2. 一个 milestone 尽量对应一个 PR。
3. 每个 PR 必须补充或更新相关文档。
4. 改动涉及 env、deployment、security、observability、API 行为时，必须同步更新 docs。
5. 涉及安全边界的改动必须覆盖 allowed 与 denied 两类测试。
6. host-agent production 行为默认应 deny-by-default。
7. 不允许引入任意 shell passthrough。
8. Docker、systemd、cron、file 操作必须继续使用结构化 API 与 allowlist/safe-root 约束。
9. 每个 milestone 完成后更新 `docs/roadmap.md`。
10. 每个 PR 结束前运行完整质量门禁。

## Required Quality Gates

```bash
make lint
make test
```

Backend:

```bash
cd backend
go test ./...
```

Core-agent:

```bash
cd core-agent
cargo fmt --all -- --check
cargo test
```

Frontend:

```bash
cd frontend
npm ci
npm run test
npm run build
```

Proto contract:

```bash
make proto-go
git diff --exit-code -- backend/internal/grpcclient/pb/proto/agent/v1/
```

---

# Milestone P3-0: Stabilization Gate

## Goal

证明当前 main 分支可构建、可测试、可启动、可冒烟，并把这个状态固化为后续所有 PR 的最低质量门禁。

## Tasks

### 1. 运行完整本地质量检查

```bash
make lint
make test
```

### 2. 运行分模块测试

```bash
cd backend && go test ./...
cd ../core-agent && cargo fmt --all -- --check && cargo test
cd ../frontend && npm ci && npm run test && npm run build
```

### 3. 验证 proto 生成文件没有漂移

```bash
make proto-go
git diff --exit-code -- backend/internal/grpcclient/pb/proto/agent/v1/agent.pb.go backend/internal/grpcclient/pb/proto/agent/v1/agent_grpc.pb.go
```

### 4. 验证 compose mode

```bash
make up
curl -f http://127.0.0.1:8080/health
curl -f http://127.0.0.1:8080/ready
make down
```

### 5. 验证 host-agent mode

```bash
make up-host-agent
curl -f http://127.0.0.1:8080/health
curl -f http://127.0.0.1:8080/ready
make down-host-agent
```

如果当前环境无法跑 host-agent mode，必须在报告中明确写出原因、缺失依赖、复现命令和后续验证环境要求。

### 6. 新增稳定性报告

创建：

```text
docs/p3-stabilization-report.md
```

内容必须包含：

```text
- 测试日期
- commit sha
- OS/runtime versions
- commands run
- pass/fail result
- fixed failures
- unresolved risks
- compose smoke evidence
- host-agent smoke evidence 或无法执行原因
```

### 7. 更新 roadmap

更新：

```text
docs/roadmap.md
docs/roadmap.zh-CN.md
```

加入 P3-0 状态。

## Acceptance Criteria

```text
- CI green
- make lint passes
- make test passes
- proto generated files are current
- compose smoke documented
- host-agent smoke documented or explicitly marked unavailable with reason
- docs/p3-stabilization-report.md exists
- docs/roadmap.md and docs/roadmap.zh-CN.md updated
- no new product feature added in this milestone
```

---

# Milestone P3-1: Alert Delivery & Operational Governance

## Goal

把当前 no-op observability baseline 变成可落地的生产告警治理体系。

## Tasks

### 1. 定义告警归属

新增或更新：

```text
docs/alerting-runbook.md
docs/alerting-runbook.zh-CN.md
```

内容必须包含：

```text
- warning alert owner
- critical alert owner
- paging vs non-paging policy
- escalation path
- dedup window
- inhibition rules
- rollback procedure
- synthetic alert test procedure
```

### 2. 配置 production receiver template

保留 local/dev no-op receiver，同时提供 production config 示例。

检查并完善：

```text
deploy/observability/alertmanager/alertmanager.production.example.yml
scripts/observability/generate-alertmanager-config.ps1
```

### 3. 生成生产配置并验证

```powershell
pwsh -File ./scripts/observability/generate-alertmanager-config.ps1 ...
pwsh -File ./scripts/observability/validate-config.ps1
```

### 4. 验证 synthetic alert delivery

```powershell
pwsh -File ./scripts/observability/alertmanager-smoke.ps1
```

### 5. 调整告警阈值

重点检查：

```text
- backend availability SLO
- backend p95 latency warning/critical
- core-agent gRPC error ratio
- backend agent transport error
- in-flight request pressure
```

### 6. 文档化运行手册

Runbook 必须回答：

```text
- 收到 SnowPanelBackendDown 怎么处理
- 收到 SnowPanelCoreAgentMetricsDown 怎么处理
- 收到 backend p95 latency high 怎么处理
- 收到 core-agent gRPC error ratio high 怎么处理
- 如何从 X-Request-ID 查 backend log 与 core-agent log
- 如何通过 Jaeger 查 trace
- 如何临时 silence
- 如何回滚错误的 alert config
```

## Acceptance Criteria

```text
- production Alertmanager receiver template exists
- warning/critical routing documented
- synthetic alert smoke passes
- validate-config.ps1 passes
- runbook covers triage, silence, escalation, rollback
- local dev no-op receiver still works
```

---

# Milestone P3-2: Backend ↔ Core-Agent Trust Boundary

## Goal

不能只依赖网络隔离保护 core-agent。host-agent 是真实机器操作入口，backend 与 core-agent 之间必须有可配置的认证边界。

## Preferred Design

```text
Preferred: mTLS
Fallback: shared token in gRPC metadata
```

可以先实现 token mode，保留 mtls mode 设计文档和接口位置。

## Tasks

### 1. 新增配置项

Backend env:

```text
BACKEND_AGENT_AUTH_MODE=none|token|mtls
BACKEND_AGENT_SHARED_TOKEN=
BACKEND_AGENT_TLS_CA_FILE=
BACKEND_AGENT_TLS_CERT_FILE=
BACKEND_AGENT_TLS_KEY_FILE=
```

Core-agent env:

```text
CORE_AGENT_AUTH_MODE=none|token|mtls
CORE_AGENT_SHARED_TOKEN=
CORE_AGENT_TLS_CA_FILE=
CORE_AGENT_TLS_CERT_FILE=
CORE_AGENT_TLS_KEY_FILE=
```

### 2. core-agent gRPC interceptor

实现：

```text
- auth mode = none: 保持当前 dev 行为
- auth mode = token: 检查 metadata 中的 token
- missing token: reject
- wrong token: reject
- correct token: allow
- logs must include request_id
- logs must never include token value
```

建议 metadata key：

```text
x-snowpanel-agent-token
```

### 3. backend gRPC client 注入 metadata

实现：

```text
- backend config load auth mode
- token mode 自动附加 x-snowpanel-agent-token
- auth failure 映射成稳定 backend error
- frontend error hint 显示 agent authentication failed / agent unavailable 区别
```

### 4. 测试

Core-agent tests:

```text
- none mode allows request
- token mode rejects missing token
- token mode rejects wrong token
- token mode allows correct token
- token is redacted from logs/errors
```

Backend tests:

```text
- token metadata attached
- missing backend token config in production fails fast
- agent auth error mapped correctly
```

### 5. 文档

更新：

```text
docs/deployment.md
docs/deployment.zh-CN.md
docs/security.md
docs/security.zh-CN.md
deploy/core-agent/systemd/core-agent.env.example
.env.example
```

必须说明：

```text
- local dev 可用 none
- production 推荐 token 或 mtls
- token rotation procedure
- 50051 仍必须限制在可信网络
- 不允许把 core-agent gRPC 暴露到公网
```

## Acceptance Criteria

```text
- core-agent can reject unauthenticated gRPC calls
- backend can authenticate to core-agent
- local dev remains simple
- production docs recommend auth enabled
- tests cover allow and deny paths
```

---

# Milestone P3-3: Secrets & Settings Hardening

## Goal

解决 sensitive settings/secrets at rest，并降低 secret 泄露风险。

## Tasks

### 1. SystemSetting 加密

设计加密 payload：

```json
{
  "version": 1,
  "algorithm": "AES-256-GCM",
  "nonce": "base64",
  "ciphertext": "base64"
}
```

新增 env：

```text
SNOWPANEL_ENCRYPTION_KEY=
SNOWPANEL_ENCRYPTION_KEY_ID=
```

### 2. 加密服务

Backend 新增：

```text
backend/internal/security/encryption.go
backend/internal/security/encryption_test.go
```

要求：

```text
- encrypt/decrypt roundtrip
- wrong key fails safely
- empty key in production fails if encrypted settings are used
- encrypted values never stored plaintext
```

### 3. Secret redaction

覆盖：

```text
- audit request summaries
- task metadata
- backend logs
- frontend error display
- config validation errors
```

Redaction keys：

```text
password
passwd
token
secret
key
credential
authorization
cookie
set-cookie
```

### 4. Production startup validation

生产环境必须 fail fast：

```text
- APP_ENV=production 且 JWT_SECRET 弱/空
- BOOTSTRAP_ADMIN=true 且 DEFAULT_ADMIN_PASSWORD 弱/空
- encrypted settings 存在但 SNOWPANEL_ENCRYPTION_KEY 缺失
- agent auth mode 开启但 token/cert 缺失
```

### 5. Token storage decision

新增：

```text
docs/security-token-storage-decision.md
```

内容：

```text
- 当前 persisted frontend bearer/refresh token 的风险
- 暂不迁移 httpOnly cookie 的原因
- 迁移 httpOnly cookie + CSRF 需要的前置条件
- 未来迁移计划
```

## Acceptance Criteria

```text
- sensitive SystemSetting values encrypted at rest
- encryption tests pass
- redaction tests pass
- production validation fails unsafe config
- token storage decision documented
```

---

# Milestone P3-4: Durable Task Worker

## Goal

将当前 goroutine async task baseline 升级为 DB-backed durable worker，避免 backend 重启导致任务孤儿、重复执行或状态不一致。

## Tasks

### 1. DB schema migration

新增字段：

```sql
ALTER TABLE tasks ADD COLUMN locked_by TEXT;
ALTER TABLE tasks ADD COLUMN locked_until TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN attempt INT NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN max_attempts INT NOT NULL DEFAULT 1;
ALTER TABLE tasks ADD COLUMN idempotency_key TEXT;
ALTER TABLE tasks ADD COLUMN next_run_at TIMESTAMPTZ;
```

新增索引：

```sql
CREATE INDEX idx_tasks_claim ON tasks(status, next_run_at, locked_until);
CREATE UNIQUE INDEX idx_tasks_idempotency_key ON tasks(idempotency_key) WHERE idempotency_key IS NOT NULL;
```

### 2. Task repository claim API

实现：

```text
ClaimNextTask(workerID, leaseDuration)
HeartbeatTask(taskID, workerID, leaseDuration)
CompleteTask(taskID, workerID, result)
FailTask(taskID, workerID, error, retryPolicy)
ReleaseStaleTasks(now)
```

要求：

```text
- 使用 transaction
- 使用 row lock 或 atomic update
- 保证同一 task 同时只有一个 worker 执行
```

### 3. Worker loop

新增配置：

```text
TASK_WORKER_ENABLED=true
TASK_WORKER_ID=<hostname-or-random>
TASK_WORKER_CONCURRENCY=2
TASK_WORKER_LEASE_DURATION=30s
TASK_WORKER_POLL_INTERVAL=2s
TASK_WORKER_MAX_ATTEMPTS=3
```

实现：

```text
- claim pending task
- heartbeat lease
- execute operation
- finish success/failed/canceled
- recover stale running task
- graceful shutdown
```

### 4. Retry policy

实现：

```text
- retryable agent transport errors
- non-retryable validation errors
- exponential backoff
- max attempts
```

### 5. Cancellation propagation

要求：

```text
- cancel pending task: terminal canceled
- cancel running task: mark cancel requested
- worker checks cancel before/after agent operation
- where possible, agent call receives context cancellation
```

### 6. Metrics

新增 metrics：

```text
snowpanel_tasks_queue_depth
snowpanel_tasks_running
snowpanel_tasks_completed_total{type,status}
snowpanel_tasks_duration_seconds{type,status}
snowpanel_task_worker_claims_total{outcome}
```

### 7. Tests

必须覆盖：

```text
- claim only once
- stale lease recovery
- backend restart simulation
- failed task retry
- max attempts reached
- cancellation before start
- cancellation while running
- idempotency key duplicate
```

## Acceptance Criteria

```text
- backend restart does not orphan tasks
- at most one worker executes one task
- retry/cancel behavior deterministic
- task logs remain auditable
- task metrics visible under /metrics
```

---

# Milestone P3-5: Host-Agent Least Privilege

## Goal

让 host-agent production 默认更保守，避免真实机器操作能力扩大 blast radius。

## Tasks

### 1. Production deny-by-default validation

Core-agent production mode 下增加校验：

```text
- CORE_AGENT_AUTH_MODE must not be none
- CORE_AGENT_SERVICE_WHITELIST must be non-empty if service manage is enabled
- CORE_AGENT_CRON_ALLOWED_COMMANDS must be explicitly configured
- CORE_AGENT_ALLOWED_ROOTS must not include / unless override flag enabled
- CORE_AGENT_HOST=0.0.0.0 requires auth mode enabled
```

新增 override：

```text
CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=false
```

### 2. Feature gates

新增：

```text
CORE_AGENT_ENABLE_FILE_OPS=true
CORE_AGENT_ENABLE_SERVICE_OPS=true
CORE_AGENT_ENABLE_DOCKER_OPS=true
CORE_AGENT_ENABLE_CRON_OPS=true
```

Production 中允许关闭某类操作。

### 3. Systemd unit hardening

检查并强化：

```text
deploy/core-agent/systemd/*.service
```

建议项：

```text
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict or full
ProtectHome=read-only where possible
ReadWritePaths=<allowed roots>
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
CapabilityBoundingSet=<minimal set>
```

如果某些 hardening 与 Docker/systemd 操作冲突，必须写清楚 tradeoff。

### 4. Startup warnings/errors

对以下配置输出明确错误或警告：

```text
- allowed roots too broad
- service whitelist empty
- cron allowlist uses defaults in production
- metrics exposed on non-loopback without warning
- gRPC exposed on non-loopback without auth
```

### 5. Docs

更新：

```text
docs/deployment.md
docs/deployment.zh-CN.md
docs/security.md
docs/security.zh-CN.md
deploy/core-agent/systemd/README.md
```

必须包含 production checklist：

```text
- gRPC bind address
- firewall
- agent auth
- allowed roots
- service whitelist
- cron command allowlist
- Docker socket risk
- metrics endpoint exposure
- backup before destructive operations
```

## Acceptance Criteria

```text
- unsafe production config fails fast unless explicitly overridden
- feature gates work
- systemd hardening documented
- tests cover config validation
```

---

# Milestone P3-6: Audit Retention, Export, and Forensics

## Goal

让敏感操作审计支持真实排障、追责、导出和保留策略。

## Tasks

### 1. Audit filters

Backend audit API 支持：

```text
- time range
- user_id
- username
- module
- action
- target_type
- target_id
- success
- result_code
- request_id
- trace_id
```

### 2. Audit export

新增 endpoint：

```text
GET /api/v1/audit/logs/export?format=csv
GET /api/v1/audit/logs/export?format=jsonl
```

要求：

```text
- permission: audit.export
- streaming response
- pagination-safe
- redacts sensitive fields
```

### 3. Audit retention

新增配置：

```text
AUDIT_RETENTION_DAYS=180
AUDIT_EXPORT_MAX_ROWS=100000
```

新增 cleanup job：

```text
- dry run mode
- archive before delete option
- audit cleanup itself must be audited
```

### 4. UI improvements only for audit usability

允许小范围 UI 改动：

```text
- filters
- detail drawer
- copy request_id
- export button
- link to task detail if task_id exists
```

### 5. Tests

覆盖：

```text
- filter query correctness
- export CSV format
- export JSONL format
- permission denied without audit.export
- redaction in export
- retention dry run
```

## Acceptance Criteria

```text
- operators can answer who did what, when, from where, and whether it succeeded
- audit export works for large datasets
- secrets are never exported
- retention policy documented
```

---

# Milestone P3-7: Backup and Restore Foundation

## Goal

在继续扩展 server panel 功能前，先建立 Postgres 和关键配置的备份/恢复基础。

## Tasks

### 1. Define backup scope

支持：

```text
- Postgres database dump
- SnowPanel app metadata
- observability config snapshot
- agent config template snapshot
```

暂不支持或需明确标记：

```text
- raw secrets export
- arbitrary filesystem backup
- Docker volume backup
- remote object storage
```

### 2. Backup model/service

基于已有 `Backup` model，实现：

```text
- create backup task
- backup status
- checksum
- size
- storage type local
- retention
- audit log
```

### 3. Backup task integration

接入 durable task worker：

```text
TaskTypeBackupCreate
TaskTypeBackupVerify
```

### 4. Restore drill docs

新增：

```text
docs/restore-drill.md
docs/restore-drill.zh-CN.md
```

必须包含：

```text
- fresh machine restore
- restore Postgres
- rotate secrets
- start backend/frontend
- start host-agent
- verify /health
- verify /ready
- login verification
- audit verification
```

### 5. Tests

覆盖：

```text
- backup metadata creation
- checksum validation
- failed backup marks task failed
- retention cleanup
- restore doc command sanity
```

## Acceptance Criteria

```text
- backup can be created and verified
- restore drill is documented
- backup operation is audited
- destructive future modules can depend on backup foundation
```

---

# Milestone P4: Feature Expansion From Existing Models

P4 只有在 P3 主要安全、治理、备份、任务可靠性完成后再启动。

## P4-1 Host Inventory

### Goal

把单一 agent target 升级成可治理的 host inventory。

### Tasks

```text
- Host CRUD
- Agent registration
- Agent heartbeat
- Agent version/capabilities
- Per-host allowed roots
- Per-host service whitelist
- Per-host cron allowlist
- Host health summary
- Host-level audit filters
```

## P4-2 Website Management

### Goal

基于已有 Website/WebsiteDomain model 增加网站管理。

### Tasks

```text
- Website CRUD
- domain binding
- root path validation
- runtime metadata
- Nginx/Caddy integration design
- backup before destructive ops
- audit all changes
```

## P4-3 Database Management

### Goal

基于已有 DatabaseInstance/Database model 增加数据库管理。

### Tasks

```text
- DB instance CRUD
- encrypted credentials
- connectivity test
- database list/create/delete
- least-privilege DB user docs
- backup before delete
- audit all changes
```

## P4-4 Backup UI

### Goal

在 P3 backup foundation 基础上提供可操作 UI。

### Tasks

```text
- backup list/detail
- create backup
- verify backup
- download metadata
- retention policy UI
- restore drill link
```

## P4-5 Plugin Framework

### Goal

谨慎引入 plugin system，避免扩大攻击面。

### Tasks

```text
- plugin manifest schema
- permission model
- signature/checksum
- sandbox policy
- install/enable/disable audit
- rollback/uninstall
```

---

# 首个可执行 Issue

## Title

```text
P3-0 Stabilization Gate for SnowPanel
```

## Body

```text
Context:
SnowPanel has completed P2 observability and prototype cleanup. Before adding new pages or operational modules, establish a production-hardening baseline.

Scope:
1. Run all local quality gates:
   - make lint
   - make test
   - backend go test ./...
   - core-agent cargo fmt --all -- --check && cargo test
   - frontend npm ci && npm run test && npm run build
2. Run compose smoke:
   - make up
   - curl /health and /ready
   - make down
3. Run host-agent smoke if environment supports it:
   - make up-host-agent
   - curl /health and /ready
   - make down-host-agent
4. Verify proto contract:
   - make proto-go
   - git diff --exit-code for generated Go proto files
5. Fix any failures without changing product behavior.
6. Add docs/p3-stabilization-report.md with command evidence and unresolved risks.
7. Update docs/roadmap.md and docs/roadmap.zh-CN.md to add P3-0 status.

Acceptance criteria:
- CI green.
- make lint and make test pass.
- Proto generated files are current.
- Compose smoke documented.
- Host-agent smoke documented or explicitly marked unavailable with reason.
- No new product features in this PR.
```

---

# Recommended Execution Order

```text
1. P3-0 Stabilization Gate
2. P3-1 Alert Delivery & Operational Governance
3. P3-2 Backend ↔ Core-Agent Trust Boundary
4. P3-3 Secrets & Settings Hardening
5. P3-4 Durable Task Worker
6. P3-5 Host-Agent Least Privilege
7. P3-6 Audit Retention, Export, and Forensics
8. P3-7 Backup and Restore Foundation
9. P4-1 Host Inventory
10. P4-2 Website Management
11. P4-3 Database Management
12. P4-4 Backup UI
13. P4-5 Plugin Framework
```

---

# Definition of Done for P3

```text
P3 is done only when:
- full test suite and CI are stable
- production alert routing is documented and validated
- backend-core-agent authentication exists
- production config fails unsafe defaults
- sensitive settings can be encrypted at rest
- task execution is durable across backend restarts
- host-agent has least-privilege production guidance
- audit export and retention exist
- backup creation and restore drill are documented
```
