请作为接手 SnowPanel 的 agent，优先按“主链路闭环 > 安全收口 > 权限模型 > 测试补齐”的顺序推进，不要先做 UI 美化，也不要先加新页面。

【项目现状判断】
这是一个“骨架已经搭起来，但宿主机控制主链路还没真正打通”的原型项目。
最大问题不在页面数量，而在以下 4 点：
1. backend -> core-agent 的 gRPC 还是占位实现，导致 Dashboard / Files / Services / Docker / Cron 这条链路名义上存在、实际上没闭环。
2. core-agent 被放在普通容器里运行，但它的实现却依赖宿主机级 systemctl / crontab / Docker socket，这和当前 compose 拓扑不匹配。
3. 安全仍然是开发态：默认 admin / 默认 JWT secret / 关键内部服务端口默认暴露。
4. cron.manage 现在实质上等于“可计划执行任意 shell 命令”，和“避免任意命令执行”的目标冲突。

【先做的事（P0）】

P0-1：把 backend 的 gRPC 客户端从占位实现改成真实实现
- 涉及文件：
  - backend/internal/grpcclient/agent_client.go
  - backend/cmd/server/main.go
  - core-agent/src/api/grpc_server.rs
  - proto/*
- 要求：
  - 用 proto 生成的 client/server 类型替换 backend 里手写的占位 struct/interface。
  - 真正连上 core-agent，完成：
    - GetSystemOverview
    - GetRealtimeResource
    - List/Read/Write/Mkdir/Delete files
    - List/Start/Stop/Restart services
    - Docker list/start/stop/restart/images
    - Cron list/create/update/delete/enable/disable
  - 做统一错误映射：gRPC error / proto error / HTTP error code 要可追踪。
  - 加最少一条 backend + agent 的集成测试，不要只保留 unit test。
- 验收标准：
  - backend 启动后，`/api/v1/dashboard/summary` 不再返回 not implemented。
  - Files / Services / Docker / Cron 至少各有 1 条 happy path 集成测试。
  - 删除 backend 里现有的 `ErrNotImplemented` 路径。

P0-2：重新定义 core-agent 的运行方式，不要继续“普通容器里控制宿主机”
- 涉及文件：
  - docker-compose.yml
  - core-agent/Dockerfile
  - core-agent/src/process/systemd_service.rs
  - core-agent/src/cron/service.rs
  - core-agent/src/docker/service.rs
- 要求：
  - 二选一并落地：
    A. 推荐：core-agent 改为宿主机 systemd service 部署，backend 通过内网 gRPC 访问；
    B. 备选：保留容器，但明确挂载 /var/run/docker.sock、需要的宿主机路径、必要 namespace/capabilities，并把风险写清楚。
  - 当前 compose 里 core-agent 没有 docker.sock、也不是 host systemd/crontab 环境，服务管理/cron/容器管理从设计上就不成立；必须先修正运行形态。
  - 50051 不要默认暴露到宿主机公网可达面。
- 验收标准：
  - 在真实目标环境里可以成功：
    - 读取宿主机 system info
    - 列出并操作宿主机 Docker container
    - 列出并操作宿主机 systemd service
    - 读写目标 cron
  - 给出明确部署文档：dev 和 prod 两套运行方式。

P0-3：先把开发态默认凭据和端口暴露收口
- 涉及文件：
  - .env.example
  - docker-compose.yml
  - frontend/src/pages/LoginPage.tsx
  - backend/internal/service/auth_service.go
- 要求：
  - 移除/禁止默认 `admin / admin123456` 作为长期可登录凭据。
  - 首次启动可以 bootstrap admin，但必须：
    - 仅首次生效
    - 强制改密
    - 生产环境必须显式提供强密码或随机生成
  - `JWT_SECRET=change-me-in-production` 不能作为默认生产可运行值。
  - Postgres / Redis / core-agent 端口默认不要对宿主机暴露；只保留 frontend/backend 必要端口。
  - 登录页不要预填生产危险默认密码。
- 验收标准：
  - 无用户时可初始化首个管理员，但不能长期沿用弱口令。
  - dev/prod 配置分离，prod 启动时若密钥弱或未配置应直接 fail fast。
  - compose 默认对外只暴露必须端口。

P0-4：重做 cron 权限模型，禁止“任意命令调度”
- 涉及文件：
  - core-agent/src/cron/service.rs
  - backend/internal/service/cron_service.go
  - backend/internal/api/router.go
- 要求：
  - 当前 `validate_command()` 只限制换行和长度，等于允许具备 `cron.manage` 的用户计划执行任意命令，这个要收掉。
  - 改为以下任一方案：
    A. 只允许执行预注册 job（如 backup/logrotate/cleanup）；
    B. 只允许执行白名单脚本目录中的脚本，且参数单独校验；
    C. 至少引入 command allowlist + shell metacharacter 禁止 + 审计明细。
  - 把文档里“无任意命令执行 API”和实际能力对齐。
- 验收标准：
  - 不能再通过 cron API 注入任意 shell 命令。
  - 有安全回归测试，覆盖危险字符、管道、重定向、子命令等场景。
  - 审计日志里能明确记录 cron 任务模板/脚本/参数。

【第二批（P1）】

P1-1：把权限模型从“按用户名硬编码”升级成真实 RBAC
- 涉及文件：
  - backend/internal/service/auth_service.go
  - backend/internal/middleware/permission.go
  - backend/internal/model/*
  - backend/migrations/*
- 要求：
  - 不要再用 `username == "admin"` 直接给超级权限。
  - 增加 roles / permissions / user_roles 等表。
  - token claims 从 DB 权限生成。
  - 支持禁用用户、角色变更后 session 失效/重签。
- 验收标准：
  - “admin 账号名”不再是权限判断条件。
  - 新建 operator 角色后，可通过 DB 配置权限而不是改代码。

P1-2：让前端权限感知和 session 管理真正成立
- 涉及文件：
  - frontend/src/routes/ProtectedRoute.tsx
  - frontend/src/layouts/AppLayout.tsx
  - frontend/src/store/auth-store.ts
  - frontend/src/lib/http.ts
  - frontend/src/api/auth.ts
- 要求：
  - 页面守卫不能只看 token 是否存在，启动时要做一次 `getMe()` / session 校验。
  - 菜单按权限动态展示，不要给无权限用户展示所有模块入口。
  - 401 时不仅清 token，还要有统一重定向和提示。
  - 优先考虑 refresh token + session revocation；至少为 token 过期/失效做完整处理。
  - 评估是否把 token 从 localStorage 改成更安全的 httpOnly cookie 方案。
- 验收标准：
  - 非 admin 用户不会看到无权限模块入口。
  - 刷新页面后能正确恢复合法 session，非法/过期 session 会被清理并跳转登录页。

P1-3：把“异步任务”从 demo 变成真正的后台作业框架
- 涉及文件：
  - backend/internal/service/task_service.go
  - backend/internal/api/router.go
  - frontend/src/pages/TasksPage.tsx
- 要求：
  - 当前任务系统只有 `CreateDemoTask`，请接入真实场景：
    - 备份
    - 大文件操作
    - 服务批量动作
    - 容器镜像拉取/重启
  - 支持取消、失败重试、进度、日志流。
- 验收标准：
  - 至少 1~2 个真实操作通过 task 框架异步执行，而不是 demo mock。

P1-4：把文件模块补到“能用于真实运维”的程度
- 涉及文件：
  - frontend/src/pages/FilesPage.tsx
  - core-agent/src/file/service.rs
- 要求：
  - 现在读文件固定 1MB、文本扩展名白名单较硬、无下载/上传/重命名/权限信息/二进制处理。
  - 先补最实用的：
    - 下载
    - 上传
    - 重命名
    - 明确的二进制文件提示
    - 大文件分块/分页读取
    - 更细的错误提示
- 验收标准：
  - 不会因为大文件/二进制文件把 UI 直接搞坏。
  - 文件操作失败时能给出可理解的原因。

【第三批（P2）】

P2-1：补齐测试矩阵，不要只停留在零散 unit test
- 现状判断：
  - 已有一些点状测试，但覆盖面明显不够，尤其缺 backend+agent+db 的 integration 和前端 e2e。
- 要求：
  - 增加：
    - proto contract tests
    - backend + core-agent + postgres 的 integration tests
    - auth / permission / path traversal / cron 安全回归测试
    - 前端登录 + 文件浏览 + 权限隐藏的 e2e
- 验收标准：
  - CI 至少覆盖：build、lint、unit、integration、关键 e2e smoke。

P2-2：补齐生产化观测能力
- 涉及方向：
  - request id 串联
  - metrics
  - tracing
  - health/readiness
  - audit 检索维度
- 要求：
  - 现在 access log 有基础日志，但还不够生产排障。
  - 给 backend/core-agent 增加 Prometheus metrics 或 OTel 方案。
- 验收标准：
  - 能定位一次用户操作跨 frontend/backend/agent 的链路。

P2-3：清理“原型痕迹”和重复逻辑
- 涉及点：
  - 各 service 层里那些 no-op `emitAudit` 可以删除或统一接入真正审计，避免 handler 记录一套、service 预留一套。
  - 把 docs 里“未来计划”和当前已实现能力重新对齐。
- 验收标准：
  - 代码里不再保留误导性的占位逻辑或重复抽象。

【执行顺序建议】
1. 先做 P0-2（确定 agent 的真实运行形态）
2. 再做 P0-1（打通真实 gRPC）
3. 然后做 P0-3 / P0-4（安全收口）
4. 再做 P1-1 / P1-2（RBAC + session）
5. 最后做 P1-3 / P1-4 / P2（真实任务、文件能力、测试与观测）

【不要先做的事】
- 不要先改配色/组件库/动画。
- 不要先扩页面数量。
- 不要先做“品牌官网式 README 美化”。
- 不要在主链路没闭环前就宣称可用于宿主机运维。

【一句话结论】
这个仓库“有继续做下去的价值”，但当前最值得改进的不是功能广度，而是：
“把 backend ↔ core-agent ↔ 宿主机 这条控制链真正跑通，并把默认高危入口全部收口”。