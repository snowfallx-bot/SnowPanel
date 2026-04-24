【规则】由于本项目为多agent合作，请你在每次任务执行结束后在此文件写入：

- 本轮会话的改动
- 你的commit摘要
- 你希望接下来的AI做什么

新对话结束后，你允许覆写此文件的内容，即改动缓存仅留一条。

请在写入时标注你的模型标识（如：gpt-5.3-codex；gpt-5.4；gemini-3.1-pro等）

下面是改动正文：

============

本轮是根据你贴回来的 `compose-smoke` 失败日志做针对性修复。失败根因不是 smoke 脚本本身，而是后端/agent 在 CI 容器环境下的两个真实兼容性问题。

本次核心完成项

1. 修复 PostgreSQL migration 中的保留字问题：
   - 修改 `backend/migrations/0001_init_schema.up.sql`
   - `databases` 表中的列名从 `collation` 改为 `db_collation`
   - 根因是 PostgreSQL 16 下 `collation` 作为关键字导致 migration 初始化在 `CREATE TABLE databases` 阶段报语法错误，数据库初始化半途失败

2. 同步 GORM model 映射：
   - 修改 `backend/internal/model/schema.go`
   - `Database.Collation` 的列映射从 `column:collation` 改为 `column:db_collation`
   - 保持对外 JSON 字段仍为 `collation`，只修数据库列名，不扩大 API 影响面

3. 修复 core-agent 在无 Docker socket 时直接崩溃的问题：
   - 修改 `core-agent/src/docker/service.rs`
   - 之前 `DockerService::new()` 在 `/var/run/docker.sock` 不存在时直接返回错误，导致整个 gRPC server 启动失败，进而让 backend readiness 超时
   - 现在改成：
     - agent 启动时允许 Docker 能力处于 unavailable 状态
     - `DockerService` 内部持有 `Option<Docker>`
     - 若 socket 不存在，则 Docker 相关 RPC 返回结构化 `6000 docker unavailable` 错误
     - 但不会阻断 Health / Dashboard / Files / Auth 这条 smoke 主链路的启动和联调

本轮修改文件

- `.claude/change-cache.md`
- `backend/migrations/0001_init_schema.up.sql`
- `backend/internal/model/schema.go`
- `core-agent/src/docker/service.rs`

本地验证

已通过：
- `cd backend && go test ./...`

环境限制：
- 当前本机 shell 环境没有 Rust 工具链，`cargo test` 无法本地执行
- 因此 core-agent 的最终验证将依赖 GitHub Actions 的 `core-agent` / `compose-smoke` job

根因结论

1. `compose-smoke` 中 postgres 初始化失败：
   - 由 migration 里 `collation` 列名触发
2. `compose-smoke` 中 core-agent 持续重启：
   - 由 CI 容器环境没有 `/var/run/docker.sock`，而 agent 把 Docker 初始化失败视为致命错误触发
3. backend readiness 超时：
   - 本质上是因为 backend 依赖的 agent 没成功起来，而不是 backend 自己没启动

commit摘要

- 计划提交：`fix(ci): unblock smoke without docker socket`

希望接下来的 AI 做什么

1. 先 push 这轮修复并观察 GitHub Actions：
   - `backend`
   - `core-agent`
   - `compose-smoke`
2. 如果 `compose-smoke` 仍失败，优先看：
   - backend `/ready` 是否仍要求 agent 的 Docker 能力可用
   - smoke 脚本是否还有对数据库初始化状态的隐式假设
3. 如果 smoke 转绿，下一步继续回到 `P2-1`：
   - 补更系统的 integration
   - 或补 frontend e2e

by: claude-sonnet-4-6
