请作为接手 SnowPanel 的 agent，优先按“主链路闭环 > 安全收口 > 权限模型 > 测试补齐”的顺序推进，不要先做 UI 美化，也不要先加新页面。

更新时间：2026-04-25

【当前状态摘要】

- backend ↔ core-agent 的真实 gRPC 主链路已经打通，Dashboard / Files / Services / Docker / Cron 不再依赖占位实现。
- 推荐生产运行形态已经切到 host-agent：`core-agent` 作为宿主机 systemd service，backend/frontend/postgres/redis 仍走 compose。
- 默认高危入口已收口：生产环境强制强 `JWT_SECRET`、bootstrap admin 强密码、首次登录强制改密、内部端口默认不对宿主机暴露。
- cron 不再允许任意 shell 命令，已改成 allowlist 模板并阻止常见 shell metacharacters。
- RBAC 已落地到 DB 角色/权限模型，session 校验已能感知权限变更和用户禁用。
- 异步任务已接入真实操作，文件模块已补到下载/上传/重命名/分块读写/二进制提示。
- 主要剩余工作集中在：
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

~~P2-1：补齐测试矩阵，不要只停留在零散 unit test~~
- 已完成：
  - backend unit tests、backend + fake agent integration-style tests、cron/auth/path traversal 安全测试已稳定运行。
  - proto contract tests 已纳入 CI（`proto-contract` job）。
  - compose smoke integration 已覆盖 login / 强制改密 / refresh rotation / dashboard / files / logout 主链路。
  - frontend e2e（登录 / 文件浏览 / 权限隐藏）已纳入 CI 并通过。
  - 新增 `backend-integration` CI job，补齐 backend + core-agent + postgres 真实链路覆盖，包含 services/docker/cron/tasks/audit 多模块契约与异步任务落库校验。
  - CI 分层已形成：`compose-smoke`（基础主链路）→ `backend-integration`（后端深链路）+ `frontend-e2e`（前端端到端）。
- 当前判断：可视为完成。

P2-2：补齐生产化观测能力
- 当前已有：
  - backend `/metrics`（Prometheus）已覆盖 HTTP 与 agent RPC 计数/时延（含 `rpc/outcome/transport` 标签）
  - backend request id / access log（现已追加 `trace_id` / `span_id`）
  - health / readiness
  - core-agent tracing 日志 + 独立 `/metrics` 端点（可输出 gRPC 请求总量/时延/in-flight）
  - Prometheus 基线部署与抓取配置（`docker-compose.observability.yml` + `deploy/observability/prometheus/prometheus.yml`）
  - Prometheus 基线告警规则（backend down、agent down、p95 高延迟、agent 错误率与并发 in-flight）
  - Alertmanager 基线路由与接入点（Prometheus `alerting` + `deploy/observability/alertmanager/alertmanager.yml`）
  - OTel tracing 基线已接入：
    - backend HTTP spans + gRPC client spans
    - core-agent gRPC server spans + remote trace context 提取
    - `otel-collector -> Jaeger` 基线部署（`deploy/observability/otel-collector/config.yaml`）
  - audit logs 基础检索
  - `X-Request-ID` 已打通 backend -> gRPC metadata -> core-agent 日志（可按同一 request_id 联查）
  - 已新增/更新 `docs/observability.md` / `docs/observability.zh-CN.md`，明确 metrics + tracing 排障路径
- 仍缺：
  - tracing 落地验证：compose / host-agent 模式下的 collector、Jaeger、跨服务 trace 串联实测
  - metrics 与 alert 的生产化落地（真实通知渠道、告警去重/升级策略、SLO/SLI 阈值校准）
- 当前判断：进行中。

P2-3：清理“原型痕迹”和重复逻辑
- 当前问题：
  - 根 README / 子模块 README 仍有部分内容需要继续与当前实现对齐。
  - 仍需继续扫描并收敛少量历史原型措辞与重复说明。
- 当前进展：
  - 已清理 `backend/README.md` 中关于 gRPC transport placeholder 的过时描述。
  - 已移除 `core-agent` 中 `tail_logs_placeholder` 占位方法。
  - 已把 root README 的 observability 入口与常用命令补齐。
  - 已将 `docs/roadmap.md` / `docs/roadmap.zh-CN.md` 从初始化草案改为当前状态路线图。
  - 已修正文档中 “Redis 仅预留后续使用” 的过时描述，改为反映当前登录限流共享状态用途。
  - 已更新 `docs/development.md` / `docs/development.zh-CN.md` 的 observability 命令与测试矩阵说明。
  - 已同步 root README 中 roadmap 导航标签，不再继续标注为“草案”。
  - 已补齐 README / development 文档中的 observability `down/logs` 命令，统一到 `Makefile` 实际命令集。
  - 已统一 deployment / observability 文档术语，避免仍以 “Prometheus UI/基线” 指代整套可观测性组件。
  - 已将 deployment 文档中的 “Compose Prototype / 原型模式” 命名统一为 “Compose Mode / Compose 模式”。
  - 已补齐 `docs/api-design.md` / `docs/api-design.zh-CN.md` 的系统与运维端点说明（`/api/v1/ping`、`/health`、`/ready`、`/metrics`）。
  - 已移除前端应用壳中的 `Linux Panel Prototype` 文案，并同步 e2e 登录后页面锚点为 `SnowPanel Operations Console`。
  - 已将 `proto/README.md` 中的 `Stubs` 表述统一为 `Bindings`，避免延续原型期命名。
  - 已同步 `docs/roadmap.md` / `docs/roadmap.zh-CN.md` 措辞，替换 `placeholder` 等遗留描述并纳入最新清理进展。
  - 已为 `docs/observability.md` / `docs/observability.zh-CN.md` 增加 tracing 实测清单，明确 compose / host-agent 两种模式下的最小验证路径。
- 当前判断：进行中。

【建议剩余执行顺序】

1. 在具备 `docker` / `cargo` 的环境优先收口 `P2-2`
   - tracing 闭环实测（compose / host-agent）
   - Alertmanager 真实通知渠道与阈值校准
2. 持续推进 `P2-3`
   - 文档与代码中的历史原型措辞、重复说明清理

【不要先做的事】

- 不要先改配色/组件库/动画。
- 不要先扩页面数量。
- 不要先做“品牌官网式 README 美化”。
- 不要在 `P2` 未补齐前就把项目描述成“生产就绪”。

【一句话结论】

这个仓库已经从“主链路没打通的原型”推进到了“主链路、安全收口、RBAC、测试矩阵、真实任务/文件能力基本完成”的阶段；接下来最值得投入的方向，是在可执行环境里完成生产观测能力实测收口，并持续清理文档与原型痕迹。
