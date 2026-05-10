# 前端 Token 存储决策

语言: [English](security-token-storage-decision.md) | **简体中文**

日期：2026-05-10

## 决策

SnowPanel 在当前 P3 加固阶段继续将 access token 与 refresh token 保存在前端持久化 auth store 中。

这是一个被接受的阶段性风险，不是最终目标。未来在部署前提补齐后，浏览器会话的更优方向仍是后端签发 httpOnly secure cookie，并配套 CSRF 防护。

## 当前风险

- 一旦发生 XSS，攻击者可能读取浏览器存储中的 bearer token 与 refresh token。
- 浏览器扩展或被攻陷的本地浏览器配置也可能访问持久化 token。
- refresh token 能换取新的 access token，因此浏览器存储被攻陷时影响更高，直到后端会话控制撤销它。

## 当前补偿控制

- 后端基于数据库用户状态与 `last_login_at` 校验 token 会话状态。
- 重新登录、改密、登出、禁用用户都会撤销旧逻辑会话。
- `/auth/refresh` 会轮转 access token 与 refresh token。
- 前端受保护路由会调用 `/auth/me` 校验会话状态。
- 收到 `401` 后会清理本地认证状态并回到登录页。

## 为什么现在不迁移

- SnowPanel 仍保留简单的 Bearer token API client 工作流。
- 当前本地与反向代理部署路径在不引入 cookie domain、SameSite、可信代理耦合时更直接。
- 如果只迁移存储，而不同时补齐 CSRF、cookie domain、secure proxy header 与回归测试，会形成一半完成的安全迁移，并带来不清晰的运维行为。

## 迁移前置条件

迁移浏览器会话到 httpOnly cookie 前，需要先实现并测试：

- 后端签发 `Secure`、`HttpOnly`、`SameSite` cookie 管理 access/refresh 状态。
- 对非安全方法增加 CSRF 防护。
- 明确可信反向代理与域名配置。
- 覆盖登录、刷新、登出、改密、`401` 恢复的浏览器回归测试。
- 为非浏览器自动化保留 API client 兼容方案。

## 后续计划

1. 在 feature flag 后增加 cookie session 支持。
2. 增加 CSRF middleware 以及前端非安全请求的 CSRF 提交。
3. 在测试中并行覆盖 Bearer 与 cookie 模式。
4. 文档化生产 cookie/domain/proxy 要求。
5. 待回归测试稳定后，将浏览器部署切换到 cookie 模式。
