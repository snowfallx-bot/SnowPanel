更新时间：2026-04-27

【本次推进（P3-1 继续推进，audit host 维度已补齐）】

- 已把 `hosts` 从纯 schema 预留推进到真实 backend 能力：
  - 新增 `HostRepository` / `HostService` / `HostHandler`
  - 新增 `GET /api/v1/hosts`、`POST /api/v1/hosts`、`GET /api/v1/hosts/:id`、`POST /api/v1/hosts/:id/check`
  - 新增 `hosts.read` / `hosts.manage` 权限种子
- 已把 agent 健康检查扩成可返回 identity：
  - `proto` 新增 `AgentIdentity`
  - `HealthCheckResponse` 现在可返回 `hostname` / `version` / `capabilities`
  - core-agent 已回填这些信息，backend gRPC client 也已支持 `CheckHealthDetails()`
- 已把 backend 的 agent 调用从“固定单 target”推进到“可按 host_id 选路”：
  - 新增 `hostctx` 与 `OptionalHostSelection` middleware
  - 新增 host-aware agent client，请求带 `host_id` 时会自动路由到目标 host
  - 现有 dashboard / files / services / docker / cron / tasks 链路都可复用该上下文选路机制
- 已补齐任务系统的 host 维度基础：
  - 新创建任务会持久化 `host_id`
  - 后台异步执行与 retry 会保留原 host 目标
  - tasks 列表已支持按 `host_id` 过滤，task summary / create result 已返回 `host_id`
- 已补上前端 host 选择闭环：
  - 新增前端 `hosts` API 与 `host-store`
  - `AppLayout` 已支持加载 host 列表，并提供全局 target host selector
  - `http` client 会自动给受保护运维 API 附加 `host_id`
  - dashboard / files / services / docker / cron / tasks / audit 的 React Query key 已带上 host scope，避免跨主机缓存串台
  - logout / session 失效时会清理本地 host 选择状态
- 已补齐 audit logs 的 `host_id` 落库、筛选与展示：
  - `audit_logs` model / dto / repository / service 已新增 `host_id`
  - `recordAudit` 现在会自动继承请求上下文里的 `host_id`
  - `GET /api/v1/audit/logs` 已支持按 `host_id` 过滤
  - audit 页面已显式展示 `Host` 列，支持按当前 target host 查看对应审计
  - 已补 migration：老库可执行 `0002_audit_logs_host_id.up.sql`，新库初始化的 `0001` 也已同步包含该字段
- 已继续收口 host 生命周期与管理页面：
  - backend 新增 `PUT /api/v1/hosts/:id`、`POST /api/v1/hosts/:id/enable`、`POST /api/v1/hosts/:id/disable`
  - host create/update 会先探测 agent，并对重复 address+port 做显式校验
  - host create/update/check/enable/disable 已接入审计记录
  - frontend 新增 `/hosts` 管理页，可新增、编辑、检查、启用、禁用 host
  - 异步任务状态更新现在会持久化 `started_at`、`finished_at` 与结构化 `result`
- 已继续补齐 host 心跳自动回写的最小闭环：
  - 新增 `HOST_HEALTH_POLL_ENABLED` / `HOST_HEALTH_POLL_INTERVAL` 配置，默认关闭
  - backend 启动时可按配置启动轻量 host health poller
  - poller 会跳过 disabled host，周期调用现有 `HostService.CheckHost()`，复用 status/version/last_seen_at 回写逻辑
  - poller 不阻塞启动，失败仅记录 warning，避免单个离线 host 影响控制面

【当前判断】

- 这次完成的是 `P3-1` 的“后端控制面底座”而不是整项收口。
- 现阶段已经具备：
  - 注册一个可达的 agent host
  - 探测 host 健康并拿到 agent 版本/能力
  - 在 API 请求层用 `host_id` 切到指定 agent
  - 让异步任务不丢目标 host
  - 在前端切换 target host 并驱动主要页面按 host 维度重新取数
  - 在审计日志中按 host 粒度回看关键操作
- 现阶段还没完成：
  - mTLS / enrollment / 吊销与轮换（属于 `P3-2`）

【验证结果】

- backend：`go test ./...` 已通过。
- backend：host health poller 配置与启动链路补齐后执行 `go test ./...`，已通过。
- core-agent：`cargo fmt --all` 已通过。
- core-agent：`cargo test` 在当前机器上未能完成，阻塞原因为缺少 MSVC `link.exe`（本地 Rust toolchain 缺编译链接环境，不是本次业务代码已确认的断言失败）。
- frontend：`npm run build` 已通过。
- frontend：host 管理页补齐后执行 `npm run build`，已通过。
- frontend：`npm run test` 仍被仓库 preflight 阻塞，原因是当前 Node 版本为 `20.18.0`，低于仓库要求的 `20.19.0`。
- frontend：补装原生可选依赖后，`npx vitest` 仍被当前 Node + 上游依赖 ESM/engine 兼容性问题阻塞；当前看到的是测试运行时环境问题，不是已定位到的页面断言失败。

【长期路线图（建议作为后续多个迭代的主线）】

建议从现在开始，把后续目标按“多主机接入与控制面成型 > 凭据与配置治理 > 备份恢复闭环 > 高层模块落地 > 扩展与发布治理”这条线持续推进，不要同时铺太多面。

### P3：从“单机面板”走向“可管理多节点的控制面”

#### P3-1：把 host/agent 模型从预留表结构做成真实能力
- 目标：
  - 让 `hosts` 表真正参与业务，而不是只停留在 schema 预留。
  - backend 能管理多个 core-agent 节点，而不是默认只连一个固定 agent。
  - 建立 host 注册、心跳、版本上报、在线状态、能力声明等基础控制面数据。
  - 前端形成“主机视角”的导航与上下文切换，现有文件/服务/docker/cron/任务/审计都能绑定到具体 host。
- 建议先做：
  - backend 抽象 agent connection manager，按 host 路由 gRPC 请求。
  - core-agent 增加 identity / capabilities / version handshake。
  - host 的新增、编辑、禁用、健康检查、证书/密钥绑定。
  - 任务、审计、指标中补齐 `host_id` 维度并前端可筛选。
- 验收标准：
  - 至少 2 台 agent 可被同一个 backend 管理。
  - 主机离线、版本不兼容、权限不足时有清晰错误分型。
  - 核心页面都能按 host 粒度运行和回放审计。

#### P3-2：把 agent 信任链和远程接入安全性补齐
- 目标：
  - 不只做到“能连”，还要做到“可信地连”。
  - 为多主机场景补齐 agent 身份认证、证书轮换、最小权限与接入审批。
- 建议先做：
  - gRPC mTLS 或等价双向认证方案。
  - agent bootstrap token / enrollment token。
  - host 禁用、吊销、重新签发与轮换流程。
  - 文档化的证书/密钥轮换与失陷处置 runbook。
- 验收标准：
  - 未注册 agent 无法接入。
  - 已吊销节点无法继续调用。
  - 轮换流程有脚本、有文档、有 smoke 验证。

### P4：补齐“生产面板”最核心的配置与数据保全能力

#### P4-1：建设系统配置与密钥治理中心
- 目标：
  - 把 `system_settings` 从基础表升级成正式的配置管理能力。
  - 对 JWT、agent 凭据、第三方 webhook、备份目标等敏感配置形成统一治理。
- 建议先做：
  - 设置项分级：公开配置 / 敏感配置 / 仅启动期配置。
  - 敏感配置加密存储、脱敏展示、变更审计、变更人追踪。
  - 配置变更的热加载边界与需要重启的配置清单。
  - 最低限度的“变更前校验 + 变更后回读验证”。
- 验收标准：
  - 管理员可以在 UI/API 中安全维护常见系统配置。
  - 敏感值不以明文回显，不经授权不可导出。
  - 所有配置变更都可审计、可回溯、可定位影响范围。

#### P4-2：补齐备份与恢复闭环，先做“能恢复”再做“好看”
- 目标：
  - 把 `backups` 表对应的能力真正落地。
  - 优先实现数据库与关键配置的备份恢复，再扩展到站点/文件资源。
- 建议先做：
  - backup job 模型统一接入现有 tasks/audit。
  - 支持本地磁盘与至少一种对象存储目标。
  - 校验和、保留策略、手动恢复、恢复前确认与风险提示。
  - 恢复演练脚本和最小可行灾备文档。
- 验收标准：
  - 能从面板触发备份、查看结果、下载或推送到远端存储。
  - 能完成一次“从备份恢复到可登录、可读配置、可继续操作”的演练。
  - 失败恢复过程有完整任务日志和审计记录。

### P5：把已预留的数据域做成真正可交付的运维模块

#### P5-1：落地网站管理模块，但只做“受控能力”不做全能建站
- 目标：
  - 让 `websites` / `website_domains` 进入真实业务。
  - 聚焦站点元数据、目录绑定、域名映射、运行时模板与基础状态检查。
- 建议先做：
  - 网站列表、详情、创建、启停、绑定目录与运行时。
  - Nginx/Caddy/Apache 三选一，先收敛一个明确实现，不要三套同时铺开。
  - 域名绑定、配置生成、配置校验、reload。
  - 站点级权限、审计与最小健康检查。
- 验收标准：
  - 能通过面板创建并管理至少一种标准站点类型。
  - 配置生成和 reload 失败时可回滚或至少可定位。
  - 站点操作全部纳入权限和审计体系。

#### P5-2：落地数据库实例管理，但先做外部实例纳管，不急着自建数据库平台
- 目标：
  - 让 `database_instances` / `databases` 成为正式模块。
  - 先解决“登记、连通、查看、最小操作”，而不是一步做到 DBaaS。
- 建议先做：
  - 支持 PostgreSQL / MySQL 二选一先落一条线。
  - 实例登记、连通性测试、数据库列表、只读元信息查看。
  - 谨慎开放创建库/创建用户/改密码等高危操作，并配审计与确认。
  - 与备份模块打通数据库备份入口。
- 验收标准：
  - 至少一个数据库引擎可稳定纳管。
  - 高危操作都有显式确认、权限控制和审计记录。
  - 失败场景不会造成“状态显示成功但实际未生效”的假阳性。

### P6：把平台做成“可扩展、可升级、可运维”的长期形态

#### P6-1：设计最小可用插件机制，不要过早开放任意代码执行
- 目标：
  - 让 `plugins` 表有明确边界的实际用途。
  - 先支持声明式扩展、只读集成或受控能力扩展，不要一上来做任意脚本插件。
- 建议先做：
  - 插件元数据、启停、版本、来源、兼容性检查。
  - 明确插件权限模型、生命周期钩子和审计边界。
  - 先定义一类安全插件接口，例如只读信息采集、外部通知集成、受控任务模板扩展。
- 验收标准：
  - 插件安装/升级/禁用/卸载流程清晰。
  - 插件兼容性和权限边界可检查、可阻断。
  - 不引入绕过现有审计与 RBAC 的旁路。

#### P6-2：补齐升级、发布与运维治理
- 目标：
  - 让项目从“仓库可运行”迈向“版本可升级、变更可发布、线上可治理”。
  - 这部分优先级不如控制面/备份，但会决定长期可维护性。
- 建议先做：
  - 数据库 migration 前向/回滚策略与升级说明。
  - backend / frontend / core-agent 版本兼容矩阵。
  - 发布说明模板、破坏性变更清单、升级前检查脚本。
  - 更明确的 SLO、值班、告警接收人与事故复盘模板。
- 验收标准：
  - 至少形成一版可重复执行的升级 runbook。
  - 新版本上线前能做兼容性预检。
  - 线上问题出现时，能依靠现有文档和观测链路完成定位与回退。

【近期可直接开工的切入点】

如果下一次会话要继续推进，建议优先从下面三个入口里选一个，而不是分散做：

1. `P3-1` 多主机控制面基础设施：这是后续网站/数据库/备份按 host 扩展的共同底座。
2. `P4-2` 备份恢复 MVP：这是最容易产生真实生产价值、也最能暴露任务框架和审计体系是否够用的一条线。
3. `P4-1` 配置与密钥治理：这是把“能跑”往“可长期运维”推进时最容易欠账的一块。

【仍然建议持续推进但不单独拉成长项目的事项】

1. 在真实生产值班组织下接入最终告警目的地（paging/IM/email）并完成审批备案。
2. 按线上流量持续微调 SLO/SLI 阈值、去重窗口、升级节奏与抑制规则。
3. 若未来引入浏览器 tracing，再补一轮前后端全链路观测说明与回归脚本。
4. 随每个新模块同步补齐安全测试、集成测试、e2e 与运维文档，不要等功能堆完再补。

【不要先做的事】

- 不要先改配色/组件库/动画。
- 不要先扩页面数量。
- 不要先做“品牌官网式 README 美化”。
- 不要在值班制度、发布流程和容量评估未固化前，将项目过早表述为“全面生产就绪”。

【一句话结论】

这个仓库已经从“主链路没打通的原型”推进到了“主链路、安全、RBAC、测试矩阵、观测链路与文档收口全部完成”的阶段；后续应沿着“多主机控制面 -> 配置与密钥治理 -> 备份恢复 -> 网站/数据库模块 -> 插件与发布治理”的顺序持续推进。
