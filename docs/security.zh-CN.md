# 安全说明

语言: [English](security.md) | **简体中文**

## 安全目标

- 防止任意命令执行。
- 最小化文件/进程/Docker 操作对主机的影响范围。
- 为敏感操作保留清晰可追溯的责任归因。

## 认证与授权

- 受保护 API 使用基于 JWT 的认证。
- 路由级权限校验（`RequirePermission` 中间件）。
- 授权 claims 来自数据库中的 `roles`、`permissions` 及其关联表。
- 权限校验不再存在基于用户名的硬编码绕过。
- 仅在用户表为空时才执行管理员初始化。
- 前端在受保护路由入口通过 `getMe()` 校验会话有效性。
- 前端导航按权限动态展示，并在 `401` 时自动回到登录页。
- 在生产环境中，若 `JWT_SECRET` 为空或过弱，backend 会在启动阶段 fail fast。
- 在生产环境且 `BOOTSTRAP_ADMIN=true` 时，弱或缺失的 `DEFAULT_ADMIN_PASSWORD` 会被拒绝。
- 在开发环境中，若 `DEFAULT_ADMIN_PASSWORD` 留空，backend 会生成一次性初始化密码。
- bootstrap 管理员首次登录会标记为 `must_change_password`。
- 当 `must_change_password=true` 时，后端仅允许访问 `/api/v1/auth/me` 与 `/api/v1/auth/change-password`。
- 改密成功后会签发新的 token，并将 `must_change_password` 置为 `false`。
- 后端会基于数据库用户状态与 `last_login_at` 校验 token 会话状态。
- 用户重新登录/改密后旧 token 会失效，被禁用用户的活动会话也会失效。
- 后端还会校验 token 内 RBAC 摘要与数据库当前角色/权限是否一致，角色或权限变更后旧会话会被强制失效并要求重新登录。
- 已支持 access/refresh 双令牌，`/auth/refresh` 会轮转两个令牌并推进会话时间戳。
- `/auth/logout` 会通过轮转会话时间戳撤销当前逻辑会话。
- 当前结论（更新于 2026-05-10）：SnowPanel 暂时继续把 access/refresh token 存在前端持久化 auth store 中，不在这一阶段迁移到 httpOnly cookie。详见[前端 Token 存储决策](security-token-storage-decision.zh-CN.md)。
- 原因：当前同源代理部署、非浏览器 API 客户端，以及现有 backend Bearer Token 链路在这个方案下更直接；而数据库支撑的 session 校验、refresh rotation、首次登录强制改密、`401` 自动跳转等控制面已经满足当前会话安全目标。
- 只有在 backend 下发安全 cookie、CSRF 防护、受信任反向代理/域名策略这三者一起设计完成，并补齐浏览器与 API client 回归测试后，才重新评估 cookie 迁移。
- 登录接口已增加基于 `username + client IP` 的防爆破保护。
- 默认模式为内存（`LOGIN_ATTEMPT_STORE=memory`）；可通过 Redis 开启分布式模式（`LOGIN_ATTEMPT_STORE=redis` 且使用共享 `REDIS_*` 配置）。
- 在 `LOGIN_FAILURE_WINDOW` 内连续失败达到阈值后，会对该键执行 `LOGIN_LOCK_DURATION` 的临时锁定并返回 `429`。

## 文件安全

- Agent 要求传入绝对路径。
- 安全根目录策略将路径访问限制在允许范围内。
- 阻止危险的删除/写入/建目录目标（`/`、`/etc`、`/usr` 等）。
- 读写大小限制可通过环境变量配置。
- 生产环境下，除非显式设置 `CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=true`，否则 `core-agent` 会拒绝 `CORE_AGENT_ALLOWED_ROOTS=/`。
- 可通过 `CORE_AGENT_ENABLE_FILE_OPS=false` 关闭文件操作。

## 运行安全

- 服务操作是显式动作（`start/stop/restart`），并进行名称校验。
- Docker 操作是显式动作（`start/stop/restart/list`），不透传 shell。
- Cron 操作通过结构化模型与校验流程执行。
- Cron 调度仅允许命令模板白名单（`CORE_AGENT_CRON_ALLOWED_COMMANDS`），并拒绝 shell 元字符。
- 生产环境启用服务操作时，要求 `CORE_AGENT_SERVICE_WHITELIST` 非空。
- 生产环境启用 Cron 操作时，要求显式配置非空 `CORE_AGENT_CRON_ALLOWED_COMMANDS`。
- 可分别通过 `CORE_AGENT_ENABLE_SERVICE_OPS=false`、`CORE_AGENT_ENABLE_DOCKER_OPS=false`、`CORE_AGENT_ENABLE_CRON_OPS=false` 关闭宿主机操作类别。
- Docker socket 访问应视为等价于宿主机 root 权限；不需要容器控制的 agent 应关闭 Docker 操作。

## Backend 到 Core-Agent 信任边界

- 本地开发可以继续使用 `BACKEND_AGENT_AUTH_MODE=none` 与 `CORE_AGENT_AUTH_MODE=none`。
- 生产环境至少应启用 `token` 模式：
  - backend：`BACKEND_AGENT_AUTH_MODE=token` 与 `BACKEND_AGENT_SHARED_TOKEN=<shared secret>`
  - core-agent：`CORE_AGENT_AUTH_MODE=token` 与 `CORE_AGENT_SHARED_TOKEN=<same shared secret>`
- token 会通过 gRPC metadata key `x-snowpanel-agent-token` 发送。
- 缺失或错误 token 会被 core-agent 以 `Unauthenticated` 拒绝；backend 会映射为 `core agent authentication failed`。
- token 值不得写入日志、API 错误或文档示例中的真实值。
- 轮换 token 时，应在同一个维护窗口将新 shared token 同步部署到 core-agent 与 backend，然后重启或 reload 两端服务。
- `mtls` 模式已预留配置位置，供未来证书认证边界使用；在实现前会 fail fast。
- 即使启用 token 模式，也必须将 core-agent gRPC 端口 `50051` 限制在可信私网内，禁止暴露到公网。
- 生产环境下，除非显式设置 `CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=true`，否则 `core-agent` 会拒绝 `CORE_AGENT_AUTH_MODE=none`。
- 启动时会对非 loopback metrics 暴露、未认证的 wildcard gRPC 监听输出警告。

## 可审计性

- 审计记录包含 user id、username、IP、module、action、target、请求摘要与结果。
- 审计记录还包含 `request_id` 与 `trace_id`，便于从 backend access log 与分布式 trace 回查审计条目。
- audit log 支持按时间范围、用户、module/action、target、结果、request id、trace id 过滤。
- audit log 可通过 `GET /api/v1/audit/logs/export?format=csv|jsonl` 导出为 CSV 或 JSONL；该接口要求 `audit.export` 权限。
- 文件/服务/docker/cron/任务操作路径都已接入审计写入。
- audit request summary、audit result message 与 task log metadata 在持久化前会针对 password、token、secret、key、credential、authorization、cookie 等常见敏感字段做脱敏。
- audit export 使用已经持久化的脱敏字段，不应作为原始 secret 导出路径。

## 静态密钥安全

- backend 已新增 AES-256-GCM 加密辅助能力，用于敏感 SystemSetting 以及后续包含 secret 的记录。
- `SystemSettingService` 会在持久化前加密 `is_encrypted=true` 的值，并在读取时解密。
- 如果缺失或使用错误的 `SNOWPANEL_ENCRYPTION_KEY`，加密设置的读写会失败。
- 如果配置了无效 encryption key，或数据库中已有 encrypted settings 但未配置 key，backend 启动会 fail fast。
- 加密 payload 包含 `version`、`algorithm`、`key_id`、`nonce`、`ciphertext`。
- 启用加密设置写入前，需要将 `SNOWPANEL_ENCRYPTION_KEY` 配置为 32 字节 base64 key。
- `SNOWPANEL_ENCRYPTION_KEY_ID` 用于标识当前活跃 key，便于后续轮换。

## 错误处理

- 后端采用统一业务错误模型与稳定错误码。
- 通过 recover 中间件避免向客户端泄露原始 panic。

## 加固待办

- 对敏感配置/密钥做静态加密存储。
- 对多地域部署进一步评估跨地域共享限流状态与故障切换策略。
