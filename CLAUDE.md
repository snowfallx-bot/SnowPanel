## 0. 全局前置提示词

```text
你现在是这个项目的主程和架构师。请在现有代码基础上迭代开发，不要推翻重来。

注意，在此项目中，人类只会扮演批准执行shell指令、输入“继续”提示词的操作。你被赋予高度权限，但请自主完成故障排查、工具链使用等任务。你可以编写ci以在GitHub上运行，必要时向人类求助，人类会返回Action的运行结果。

项目目标：开发一个类似宝塔 / 1Panel 的 Linux 面板原型。
技术约束：
- 核心系统层必须使用 Rust
- 后端必须使用 Golang
- 前端使用 React + TypeScript + Vite
- 后端与 Rust Agent 使用 gRPC 通信
- 前端与后端使用 REST / WebSocket 通信
- 数据库使用 PostgreSQL
- 缓存使用 Redis

开发原则：
1. 先阅读现有项目结构，再决定修改方案
2. 不要一次生成过多内容，优先保证每一步可运行
3. 所有输出必须是可落地代码，不要只写伪代码
4. 不允许实现任意命令执行
5. 不允许把用户输入直接拼接到 shell 命令中
6. 所有危险操作都要做参数校验、权限校验、审计预留
7. controller / handler 不能堆积业务逻辑，要分层
8. 每次输出必须包含：
   - 本次目标
   - 修改/新增的文件列表
   - 关键设计说明
   - 如何运行
   - 如何验收
   - 后续建议
9. 如果发现我的要求与现有代码冲突，优先保持架构清晰，并明确说明原因
10. 除非我明确要求，否则不要擅自引入过多额外依赖

代码风格要求：
- Rust：清晰模块划分、结构化错误处理、禁止滥用 unwrap
- Go：Gin + GORM + service/repository 分层 + DTO 分离
- Frontend：组件化、hooks 分离、API 封装、状态管理清晰

输出要求：
- 优先修改真实文件内容
- 优先给出完整文件，而不是只给片段
- 若文件过长，可只输出关键部分，但必须说明其余部分如何衔接
- 不要省略关键配置文件

接下来的所有操作，即便实现小改动也提交到github，如果你认为需要的话。在你提交之后，人类会为你输入GPG Passphrase。
```

---

# 第一阶段：架构和骨架

## 1. 初始化工程骨架

```text
请先完成整个项目的初始工程骨架，要求：

目标：
- 建立 monorepo 风格目录结构
- 初始化 frontend / backend / core-agent / deploy / docs
- 所有子项目都要有最小可运行骨架
- 提供根目录 README、Makefile、.gitignore、.env.example
- 提供 docker-compose 开发环境骨架（postgres、redis）

要求：
1. backend 初始化为 Go 项目，包含基础目录：
   - cmd/server
   - internal/api
   - internal/service
   - internal/repository
   - internal/model
   - internal/dto
   - internal/config
   - internal/middleware
   - internal/grpcclient
2. core-agent 初始化为 Rust 项目，包含基础目录：
   - src/api
   - src/service
   - src/system
   - src/file
   - src/process
   - src/docker
   - src/cron
   - src/security
   - src/config
3. frontend 初始化为 React + TypeScript + Vite 项目，带基础路由和布局
4. docs 下生成 architecture.md / roadmap.md / development.md 草稿
5. 不要实现复杂业务，只做工程骨架和最小启动代码
6. 每个子项目都要能通过最小命令启动

输出：
- 项目目录树
- 所有新增文件
- 关键文件内容
- 启动方法
- 下一步建议
```

---

## 2. 设计数据库 schema 和 migration

```text
请在现有项目基础上，为这个 Linux 面板设计 PostgreSQL 数据库 schema，并生成 migration 文件。

至少包含以下表：
- users
- roles
- permissions
- role_permissions
- user_roles
- audit_logs
- system_settings
- hosts
- tasks
- task_logs
- websites
- website_domains
- database_instances
- databases
- plugins
- backups

要求：
1. 使用 PostgreSQL
2. 输出真实 migration SQL 文件
3. 为关键字段加上约束、索引、唯一键、时间戳
4. users 要支持账号状态、密码哈希、最后登录时间
5. audit_logs 要支持操作者、IP、操作类型、目标对象、结果、摘要
6. tasks / task_logs 要适合异步任务执行状态追踪
7. system_settings 用于通用键值配置
8. 解释各表之间关系
9. backend 中补充 model 定义草稿

输出：
- migration 文件列表
- 每个表设计说明
- Go model 文件
- 初始化数据库方法
- 验收方法
```

---

## 3. 设计 gRPC 协议

```text
请为 Go backend 与 Rust core-agent 设计 gRPC 协议，并生成 proto 文件。

至少包含以下服务能力：
- 获取系统概览
- 获取实时资源信息
- 列出文件
- 读取文本文件
- 写入文本文件
- 创建目录
- 删除文件
- 获取服务列表
- 启动服务
- 停止服务
- 重启服务
- 获取 Docker 容器列表
- 启动/停止 Docker 容器
- 读取 cron 任务列表

要求：
1. proto 设计要考虑未来扩展
2. 返回结构统一、清晰
3. 错误信息要有 code/message
4. 路径类操作要有安全校验预留字段说明
5. 尽量把系统操作封装成明确接口，不允许任意命令执行接口
6. 生成 proto 文件后，同时生成：
   - Rust 侧使用说明
   - Go 侧使用说明
   - backend grpc client 骨架
   - core-agent grpc server 骨架

输出：
- proto 文件完整内容
- 生成代码的命令
- Go 和 Rust 对接方式
- 下一步建议
```

---

# 第二阶段：后端最小可运行链路

## 4. 实现 Go 后端基础框架

```text
请在 backend 中实现最小可运行后端框架。

目标：
- Gin 服务可启动
- 接入基础配置加载
- 接入 GORM 数据库连接
- 提供统一响应结构
- 提供基础中间件
- 提供 health 接口
- 提供 API 路由分组

要求：
1. 使用 Gin + GORM + Viper + Zap
2. 统一响应格式：
   {
     "code": 0,
     "message": "ok",
     "data": {}
   }
3. 中间件至少包含：
   - request id
   - recover
   - access log
4. 提供 /health 和 /api/v1/ping
5. 代码结构要分层，不要把逻辑全写在 main
6. 预留 grpc client 初始化
7. 预留数据库自动迁移或 migration 执行说明

输出：
- 修改文件列表
- 关键代码
- 运行命令
- 验收方式
```

---

## 5. 实现认证与用户基础模块

```text
请实现 backend 的认证与用户基础模块。

功能：
- 初始化默认管理员
- 登录
- 获取当前用户信息
- JWT 鉴权中间件
- 密码哈希校验

要求：
1. 使用 bcrypt 或 argon2 存储密码哈希
2. 登录接口：
   POST /api/v1/auth/login
3. 当前用户接口：
   GET /api/v1/auth/me
4. JWT 中间件只保护需要登录的路由
5. DTO、model、service、repository 分层清晰
6. 统一错误码和错误返回
7. 审计日志预留接口或埋点位置
8. 默认管理员初始化逻辑只在无用户时执行

输出：
- 新增/修改文件
- API 示例
- 如何创建默认管理员
- 验收步骤
```

---

## 6. 实现 RBAC 基础能力

```text
请在现有认证模块基础上，增加 RBAC 基础能力。

目标：
- role / permission / user role 基础查询
- 路由级权限校验中间件
- 先实现最小版权限模型

要求：
1. 至少支持：
   - dashboard.read
   - files.read
   - files.write
   - services.read
   - services.manage
2. 提供权限校验中间件
3. me 接口返回当前用户角色和权限
4. 不需要先做复杂后台管理页面，但代码结构要可扩展
5. service / repository 层清晰
6. 权限不足时返回统一错误码

输出：
- 权限模型说明
- 修改文件
- 示例权限校验方式
- 验收方式
```

---

# 第三阶段：Rust Agent 最小闭环

## 7. 实现 Rust Agent 基础服务

```text
请在 core-agent 中实现最小可运行的 Rust gRPC 服务。

目标：
- Agent 可启动
- 实现健康检查
- 实现系统信息获取
- 返回主机名、OS、内核、CPU、内存、磁盘基础信息

要求：
1. 使用 tonic + tokio + tracing + sysinfo
2. gRPC server 可正常监听端口
3. 错误返回结构化
4. 禁止使用 unwrap 处理关键逻辑
5. 把系统信息逻辑放入独立 service 模块
6. 提供最小 main.rs 组织结构
7. 生成 README 说明如何启动 agent

输出：
- 文件列表
- 核心代码
- 启动方法
- 与 proto 的对接说明
- 验收方法
```

---

## 8. 打通 Go 后端与 Rust Agent

```text
请把 backend 与 core-agent 打通，实现第一个完整链路。

目标：
- backend 通过 grpc client 调用 core-agent
- 新增 dashboard summary 接口
- 返回系统摘要信息到前端可消费格式

要求：
1. 接口：
   GET /api/v1/dashboard/summary
2. backend 不能直接读取系统信息，必须通过 Rust agent 获取
3. service 层负责调用 grpc client
4. handler 层只做参数和响应处理
5. 若 agent 不可用，要返回明确错误
6. 补充必要配置项（agent 地址等）

输出：
- 修改文件列表
- 调用链说明
- 如何本地联调
- 验收方式
```

---

# 第四阶段：前端基础面板

## 9. 初始化前端布局和登录页

```text
请在 frontend 中实现基础管理面板框架。

目标：
- 登录页
- 基础后台布局
- 左侧菜单
- 顶部栏
- 路由守卫
- token 管理
- 与 backend 登录接口联调

要求：
1. 使用 React + TypeScript + Vite + Tailwind + shadcn/ui
2. 路由至少包含：
   - /login
   - /dashboard
3. 登录成功后跳转 dashboard
4. 用统一 API 请求封装
5. 使用 Zustand 或等价状态管理做 auth store
6. UI 风格偏现代运维面板
7. 页面结构和组件分层清晰

输出：
- 文件结构
- 核心组件
- 如何运行
- 联调步骤
```

---

## 10. 实现 dashboard 页面

```text
请在前端实现 dashboard 页面，并接入 backend 的 /api/v1/dashboard/summary 接口。

页面至少展示：
- 主机名
- 系统版本
- 内核版本
- CPU 使用率
- 内存使用率
- 磁盘使用率
- uptime

要求：
1. 页面采用卡片式布局
2. 使用图表或进度条展示资源占用
3. 使用 TanStack Query 管理请求状态
4. 做好 loading / error / empty 处理
5. UI 要简洁专业
6. 不要先做假数据，直接接真实接口；如接口未完成可临时加 mock 开关

输出：
- 修改文件列表
- 页面结构说明
- 运行和验收方式
```

---

# 第五阶段：文件管理

## 11. Rust 实现安全文件操作

```text
请在 core-agent 中实现文件管理能力。

目标：
- 列出目录
- 读取文本文件
- 写入文本文件
- 创建目录
- 删除文件/目录（基础版）
- 路径安全校验

要求：
1. 必须实现路径校验器，防止路径穿越
2. 必须限制危险路径访问策略，至少预留白名单根目录机制
3. 文本读取要限制文件大小
4. 写入前要校验是否为文本类文件
5. 删除操作要返回明确结果
6. 通过 gRPC 暴露文件能力
7. 所有错误都结构化返回

输出：
- 修改文件
- 安全设计说明
- 验收方法
```

---

## 12. Go 封装文件 API

```text
请在 backend 中封装文件管理 API，调用 Rust agent 文件能力。

接口至少包括：
- GET /api/v1/files/list?path=
- POST /api/v1/files/read
- POST /api/v1/files/write
- POST /api/v1/files/mkdir
- DELETE /api/v1/files/delete

要求：
1. 参数校验清晰
2. 所有操作都记录审计日志预留点
3. 接口需要权限控制
4. DTO / service / handler 分层清楚
5. 统一返回格式
6. 遇到 agent 错误要转换为可读错误响应

输出：
- 文件列表
- API 设计说明
- 联调方法
- 验收步骤
```

---

## 13. 前端实现文件管理页

```text
请在 frontend 中实现文件管理页面，并接入 backend 文件 API。

功能：
- 浏览目录
- 点击进入子目录
- 返回上级目录
- 查看文本文件
- 编辑并保存文本文件
- 新建目录
- 删除文件
- 基础确认框

要求：
1. UI 类似轻量服务器文件管理器
2. 左侧可选面包屑或路径栏
3. 文件表格展示名称、类型、大小、修改时间
4. 文本编辑器先用简单 textarea 或基础代码编辑器
5. 危险操作必须二次确认
6. loading / error 状态要完整

输出：
- 页面和组件文件
- 交互说明
- 联调步骤
- 验收方法
```

---

# 第六阶段：服务管理

## 14. Rust 实现服务管理

```text
请在 core-agent 中实现 Linux 服务管理能力。

目标：
- 列出系统服务
- 查询服务状态
- 启动服务
- 停止服务
- 重启服务

要求：
1. 基于 systemd/systemctl 做封装
2. 不能开放任意命令执行
3. 服务名称要做参数校验
4. 建议通过白名单或规则限制可操作服务
5. 返回明确状态信息
6. gRPC 接口清晰
7. 预留查看日志能力的接口

输出：
- 文件列表
- 安全设计
- 运行与验收方式
```

---

## 15. Go + 前端接入服务管理

```text
请同时完成：
1. backend 服务管理 API
2. frontend 服务管理页面

后端接口至少：
- GET /api/v1/services
- POST /api/v1/services/:name/start
- POST /api/v1/services/:name/stop
- POST /api/v1/services/:name/restart

前端页面要求：
- 列表展示服务名、状态
- 支持启动/停止/重启操作
- 危险操作确认
- 操作后刷新状态
- 显示成功/失败提示

要求：
1. 权限控制
2. 审计日志预留
3. UI 清晰
4. 结构清楚，不要把页面写成巨型组件

输出：
- 修改文件
- 调用流程
- 验收方法
```

---

# 第七阶段：Docker 管理

## 16. Rust 实现 Docker 基础能力

```text
请在 core-agent 中实现 Docker 基础管理能力。

目标：
- 获取容器列表
- 启动容器
- 停止容器
- 重启容器
- 获取镜像列表

要求：
1. 使用 Rust Docker 客户端库，如 bollard
2. 不通过 shell 调 docker 命令，优先走 API
3. 返回结构清晰，便于前端展示
4. 出错信息结构化
5. 对不存在容器有明确错误响应

输出：
- 文件列表
- 关键实现说明
- 验收方式
```

---

## 17. Go + 前端接入 Docker 管理

```text
请完成 backend Docker API 和 frontend Docker 页面。

接口至少：
- GET /api/v1/docker/containers
- POST /api/v1/docker/containers/:id/start
- POST /api/v1/docker/containers/:id/stop
- POST /api/v1/docker/containers/:id/restart
- GET /api/v1/docker/images

前端要求：
- 容器列表
- 镜像列表
- 操作按钮
- 状态刷新
- 错误提示

要求：
1. 分层清晰
2. 页面不要过度复杂
3. 优先可用性
4. 危险操作要确认

输出：
- 修改内容
- 联调方式
- 验收步骤
```

---

# 第八阶段：计划任务

## 18. 实现 cron 管理闭环

```text
请实现 Linux cron 管理的完整闭环，包括 core-agent、backend、frontend。

目标：
- 列出 cron 任务
- 新增 cron 任务
- 修改 cron 任务
- 删除 cron 任务
- 启用/禁用任务（可通过注释策略实现）

要求：
1. 不允许直接任意写 crontab 原文而无校验
2. 要对 cron 表达式做基础校验
3. 要为任务定义数据结构
4. backend 记录操作审计和任务状态
5. frontend 提供表格和表单界面
6. 先支持当前用户或指定托管策略，不要做过多复杂权限穿透

输出：
- 文件列表
- 数据结构
- 交互说明
- 验收方法
```

---

# 第九阶段：审计与任务系统

## 19. 实现审计日志

```text
请在 backend 中实现审计日志基础能力，并在已有关键操作中接入。

至少记录：
- 操作者用户 ID
- 用户名
- IP
- 操作模块
- 操作类型
- 目标对象
- 请求摘要
- 成功/失败
- 时间

要求：
1. 提供 audit service/repository
2. 在文件操作、服务管理、docker 管理、登录行为中接入
3. 提供接口：
   GET /api/v1/audit/logs
4. 支持分页
5. 前端新增审计日志页面

输出：
- 修改文件
- 接入点说明
- 验收方式
```

---

## 20. 实现异步任务基础框架

```text
请在 backend 中实现异步任务基础框架，用于后续备份、恢复、较长耗时任务。

目标：
- task / task_log 模型接入
- 提供任务创建、状态更新、日志追加能力
- 至少实现一个 demo：异步执行“系统信息采集刷新”或“模拟备份任务”

要求：
1. 状态至少包括 pending/running/success/failed
2. service 层封装清晰
3. 支持简单 worker 模式或 goroutine + DB 状态管理
4. 提供查询任务列表/详情接口
5. 前端增加基础任务列表页面或任务抽屉

输出：
- 文件列表
- 任务流转说明
- 验收步骤
```

---

# 第十阶段：工程完善

## 21. Docker Compose 与本地开发完善

```text
请完善整个项目的本地开发与容器化环境。

目标：
- 完善 docker-compose.yml
- frontend/backend/core-agent/postgres/redis 都可联动
- 补充 .env.example
- 补充各服务 Dockerfile
- 根目录 Makefile 支持常见命令

Makefile 至少包含：
- make up
- make down
- make backend
- make agent
- make frontend
- make lint
- make test

要求：
1. 优先面向开发环境
2. 配置清晰
3. 所有服务的端口和依赖关系合理
4. README 更新启动说明

输出：
- 修改文件
- 启动步骤
- 常见问题说明
- 验收方法
```

---

## 22. 增加测试和文档

```text
请为当前项目补充最小必要测试与文档。

测试要求：
- backend：auth service、middleware 基础测试
- rust：path validator、system info 测试
- frontend：登录页或 dashboard 基础渲染测试

文档要求：
- README.md
- docs/architecture.md
- docs/api-design.md
- docs/security.md
- docs/development.md
- docs/deployment.md

要求：
1. 文档要与现有代码一致
2. 测试优先覆盖关键基础模块
3. 不要求追求高覆盖率，但要真实可运行
4. 给出测试命令

输出：
- 新增/修改文件
- 测试说明
- 文档摘要
- 验收方式
```

---

# 你的实际执行顺序

* 0
* 1
* 2
* 3
* 4
* 5
* 7
* 8
* 9
* 10
* 11
* 12
* 13
* 14
* 15
* 16
* 17
* 18
* 19
* 20
* 21
* 22

也就是：

* 先骨架
* 再数据库
* 再 proto
* 再后端基础
* 再认证
* 再 agent
* 再打通
* 再前端
* 再文件
* 再服务
* 再 docker
* 再 cron
* 再审计
* 再任务系统
* 最后补工程化和测试