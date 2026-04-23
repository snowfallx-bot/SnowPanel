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
- 登录接口已增加基于 `username + client IP` 的内存防爆破保护。
- 在 `LOGIN_FAILURE_WINDOW` 内连续失败达到阈值后，会对该键执行 `LOGIN_LOCK_DURATION` 的临时锁定并返回 `429`。

## 文件安全

- Agent 要求传入绝对路径。
- 安全根目录策略将路径访问限制在允许范围内。
- 阻止危险的删除/写入/建目录目标（`/`、`/etc`、`/usr` 等）。
- 读写大小限制可通过环境变量配置。

## 运行安全

- 服务操作是显式动作（`start/stop/restart`），并进行名称校验。
- Docker 操作是显式动作（`start/stop/restart/list`），不透传 shell。
- Cron 操作通过结构化模型与校验流程执行。
- Cron 调度仅允许命令模板白名单（`CORE_AGENT_CRON_ALLOWED_COMMANDS`），并拒绝 shell 元字符。

## 可审计性

- 审计记录包含 user id、username、IP、module、action、target、请求摘要与结果。
- 文件/服务/docker/cron/任务操作路径都已接入审计写入。

## 错误处理

- 后端采用统一业务错误模型与稳定错误码。
- 通过 recover 中间件避免向客户端泄露原始 panic。

## 加固待办

- 增加 refresh token 与会话吊销机制。
- 对敏感配置/密钥做静态加密存储。
- 将登录限流与锁定从单进程内存模式扩展到分布式/共享状态。
