请作为接手 SnowPanel 的 agent，优先按“主链路闭环 > 安全收口 > 权限模型 > 测试补齐”的顺序推进，不要先做 UI 美化，也不要先加新页面。

更新时间：2026-04-24

【当前状态摘要】

- backend ↔ core-agent 的真实 gRPC 主链路已经打通，Dashboard / Files / Services / Docker / Cron 不再依赖占位实现。
- 推荐生产运行形态已经切到 host-agent：`core-agent` 作为宿主机 systemd service，backend/frontend/postgres/redis 仍走 compose。
- 默认高危入口已收口：生产环境强制强 `JWT_SECRET`、bootstrap admin 强密码、首次登录强制改密、内部端口默认不对宿主机暴露。
- cron 不再允许任意 shell 命令，已改成 allowlist 模板并阻止常见 shell metacharacters。
- RBAC 已落地到 DB 角色/权限模型，session 校验已能感知权限变更和用户禁用。
- 异步任务已接入真实操作，文件模块已补到下载/上传/重命名/分块读写/二进制提示。
- 主要剩余工作集中在：
  - `P2-1` 测试矩阵补齐
  - `P2-2` 生产观测能力
  - `P2-3` 文档与原型痕迹清理

【完成情况】

~~P0-1：把 backend 的 gRPC 客户端从占位实现改成真实实现~~
- 已完成：
  - backend 使用 proto 生成的 gRPC client 调 core-agent。
  - core-agent 已提供真实 gRPC server 实现。
  - backend 侧已有统一 agent error -> HTTP/app error 映射。
  - 已有 backend + fake agent 的 happy path 集成测试，覆盖 Dashboard / Files / Services / Docker / Cron。
- 当前判断：可视为完成。

~~P0-2：重新定义 core-agent 的运行方式，不要继续“普通容器里控制宿主机”~~
- 已完成：
  - 已提供 `docker-compose.host-agent.yml`。
  - 已提供 host systemd 部署模板与文档。
  - Ubuntu 25.10 一键安装脚本默认按 host-agent 模式部署。
  - `50051` 在默认 compose 下仅 internal expose，不默认暴露到宿主机公网。
  - dev / prod 两套运行方式文档已明确区分。
- 当前判断：可视为完成。

~~P0-3：先把开发态默认凭据和端口暴露收口~~
- 已完成：
  - 登录页不再预填危险默认密码。
  - `.env.example` 不再提供生产可直接运行的弱 `JWT_SECRET`。
  - 生产环境下弱/空 `JWT_SECRET` 会 fail fast。
  - 生产环境下 `BOOTSTRAP_ADMIN=true` 时必须提供强密码。
  - development 下可自动生成一次性 bootstrap 密码。
  - bootstrap admin 首次登录会被要求强制改密。
  - Postgres / Redis / core-agent 默认只 `expose`，不映射宿主机端口。
- 当前判断：可视为完成。

~~P0-4：重做 cron 权限模型，禁止“任意命令调度”~~
- 已完成：
  - core-agent 侧 `validate_command()` 已切到 allowlist 校验。
  - 已阻止常见 shell metacharacters、管道、重定向、子命令注入。
  - 文档与 API 说明已改成 command template key，而不是任意 shell 文本。
  - cron handler 已记录审计摘要。
  - 安全回归测试已覆盖危险命令输入。
- 当前判断：可视为完成。

~~P1-1：把权限模型从“按用户名硬编码”升级成真实 RBAC~~
- 已完成：
  - migrations 中已有 `roles` / `permissions` / `role_permissions` / `user_roles`。
  - token claims 已从 DB RBAC 生成。
  - permission middleware 走权限名校验，不再依赖 `username == "admin"`。
  - `ValidateSession()` 已校验用户状态、session issued time、RBAC checksum。
  - 用户禁用、角色权限变化后，旧 session 会失效。
- 当前判断：可视为完成。

~~P1-2：让前端权限感知和 session 管理真正成立~~
- 已完成：
  - `ProtectedRoute` 启动时会调用 `getMe()` 做 session 校验。
  - `ProtectedRoute` 会在 `getMe()` 成功前阻止受保护内容渲染，并对非鉴权失败展示可重试的 session 错误态。
  - `AppLayout` 已按权限动态展示菜单入口。
  - `AppLayout` 已对首次登录强制改密场景提供前端门禁，并在改密成功后刷新本地 session。
  - `401` 时前端会统一清理凭据、写入提示并跳转登录页。
  - refresh token 已接入，后端也支持 refresh rotation / session 校验。
  - 非 admin 用户不会看到无权限模块入口。
  - 已补前端回归测试，覆盖 `ProtectedRoute` 的 session 校验/失效/重试分支，以及 `AppLayout` 的权限导航/强制改密/退出登录分支。
  - token 存储策略已形成明确决议并写入 `docs/security.md` / `docs/security.zh-CN.md`：当前阶段继续使用前端持久化 bearer token，不把 httpOnly cookie 迁移作为发布前阻塞项。
- 当前判断：可视为完成；若未来推进 cookie 方案，归入后续认证加固迭代，而不是继续挂在本项名下。

~~P1-3：把“异步任务”从 demo 变成真正的后台作业框架~~
- 已完成：
  - 已移除 demo task 方向，当前任务系统已接入真实操作。
  - backend task service 支持真实的 docker restart / service restart。
  - 已支持取消、失败重试、进度、日志记录与详情查看。
  - 前端 `TasksPage` 已对接真实任务列表与详情。
- 当前判断：按原验收标准可视为完成。

~~P1-4：把文件模块补到“能用于真实运维”的程度~~
- 已完成：
  - 已支持下载、上传、重命名。
  - 已有明确二进制文件提示。
  - backend/core-agent 已支持大文件分块下载与上传。
  - 前端支持 preview limit 调整、下载/上传进度和更细错误提示。
  - 安全校验包含 safe roots / dangerous path / encoding / size 等错误分型。
- 当前判断：按原验收标准可视为完成。

P2-1：补齐测试矩阵，不要只停留在零散 unit test
- 当前已有：
  - backend unit tests
  - backend + fake agent integration-style tests
  - cron / auth / path traversal 等安全相关测试
  - frontend vitest 单测
  - CI workflow 基础构建
- 明显缺失：
  - proto contract tests
  - backend + core-agent + postgres 的真实 integration tests
  - 前端 e2e（登录 / 文件浏览 / 权限隐藏）
  - 更完整的 CI 分层矩阵（integration / smoke e2e）
- 当前判断：未完成。

P2-2：补齐生产化观测能力
- 当前已有：
  - backend request id
  - access log
  - health / readiness
  - core-agent tracing 日志
  - audit logs 基础检索
- 仍缺：
  - Prometheus metrics 或 OTel
  - 更完整的 frontend/backend/agent 链路串联
  - 面向生产排障的统一 tracing / metrics 方案
- 当前判断：未完成。

P2-3：清理“原型痕迹”和重复逻辑
- 当前问题：
  - `backend/README.md` 仍有关于 grpc transport placeholder 的过时描述。
  - 部分文档判断已明显落后于当前实现。
  - 代码中仍有少量占位痕迹，例如 `tail_logs_placeholder`。
- 当前判断：未完成。

【建议剩余执行顺序】

1. 先做 `P2-1`
   - 补真实 integration 和关键 e2e
2. 再做 `P2-2`
   - metrics / tracing / 统一链路观测
3. 最后做 `P2-3`
   - 文档与代码占位痕迹清理

【不要先做的事】

- 不要先改配色/组件库/动画。
- 不要先扩页面数量。
- 不要先做“品牌官网式 README 美化”。
- 不要在 `P2` 未补齐前就把项目描述成“生产就绪”。

【一句话结论】

这个仓库已经从“主链路没打通的原型”推进到了“主链路、安全收口、RBAC、前端 session 管理、真实任务/文件能力基本完成”的阶段；接下来最值得投入的方向，是补齐测试矩阵，并补上生产观测能力。
