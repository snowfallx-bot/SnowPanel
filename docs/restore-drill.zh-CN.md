# SnowPanel 恢复演练

本文档定义 P3 backup foundation 的最小恢复路径。范围刻意限制在 SnowPanel 控制面恢复，不覆盖任意主机文件或 Docker volume 恢复。

## 备份范围

P3 包含：

- Postgres database dump，包含 SnowPanel 元数据。
- 数据库中的 SnowPanel app metadata。
- `deploy/observability` 下的观测配置快照。
- `deploy/core-agent` 下的 core-agent 配置模板。
- 备份元数据：checksum、size、storage type、status 与审计记录。

P3 明确不包含：

- 原始 secrets 导出。
- 任意文件系统备份。
- Docker volume 备份。
- 远端对象存储。
- 完整主机裸机恢复。

## 恢复前输入

开始恢复前收集：

- 已验证的 Postgres dump。
- SnowPanel backup metadata 中记录的 checksum 与 size。
- 备份对应的 SnowPanel deployment commit 或 release version。
- 新生成的生产 secrets：JWT、encryption、database password、agent auth token。
- 预期的 core-agent allowed roots、service whitelist、cron allowlist 与其他 allowlist。

不要复用已经泄露或来源不明的备份内 secrets。

## 新机器恢复

1. 在新主机安装 Docker、Docker Compose、PowerShell 7、Go、Node.js、Rust，并拉取 SnowPanel 仓库。
2. 切换到备份元数据记录的 release 或 commit。
3. 基于 `.env.example` 创建新的 `.env`。
4. 恢复 observability 与 core-agent config snapshot 前，先复核主机相关路径与 endpoint。
5. 先启动 Postgres，不要立即启动 backend。

## 恢复 Postgres

命令形态示例：

```bash
psql "$DATABASE_URL" < snowpanel-postgres.dump.sql
```

恢复后确认：

- migrations 处于预期版本。
- `users`、`roles`、`permissions`、`role_permissions`、`tasks`、`task_logs`、`audit_logs`、`backups` 表存在。
- 没有在缺少 `SNOWPANEL_ENCRYPTION_KEY` 的情况下读取 encrypted setting。

## 轮换 Secrets

服务对外暴露前轮换：

- `JWT_SECRET`
- `SNOWPANEL_ENCRYPTION_KEY`
- `BACKEND_AGENT_SHARED_TOKEN`
- `CORE_AGENT_SHARED_TOKEN`
- 如果恢复时沿用了旧数据库凭据，也要轮换 database password。

如果恢复了 encrypted settings，只在解密并用新 key 重新加密所需的最短时间内保留旧 encryption key。

## 启动服务

启动 backend/frontend：

```bash
make up
```

host-agent mode：

```bash
make up-host-agent
```

## 验证健康状态

执行：

```bash
curl -f http://127.0.0.1:8080/health
curl -f http://127.0.0.1:8080/ready
```

预期结果：

- `/health` 返回 healthy。
- `/ready` 返回 ready。
- backend log 没有缺失 encryption key、unsafe production agent auth 或 database migration 错误。

## 登录验证

1. 使用 administrator 账号登录。
2. 立即修改 bootstrap 或 emergency password。
3. 确认 session 中包含预期 permissions。
4. 确认受保护页面不会出现 401/403。

## 审计验证

登录后确认：

- 新的 login 与配置操作会进入 audit logs。
- 新 audit records 带有 `request_id`。
- CSV 与 JSONL audit export 仍可使用。
- audit retention cleanup 不会自动执行，除非 operator 显式触发。

## Task 与 Backup 验证

执行非破坏性 task 检查：

- 查看 task list。
- 打开一个历史 task detail。
- 确认 task logs 可见。

backup foundation 验证：

- 确认 backup metadata rows 包含 checksum、size、status、storage type 与 resource 字段。
- 确认本地 artifacts 位于 `BACKUP_LOCAL_DIR` 下，且不在公开 Web 根目录中。
- 对恢复出的 backup artifact 重新计算 checksum，并与 metadata 比对。
- 可用 `{}` 请求体调用 `POST /api/v1/backups/:id/verify-task`，让 worker 从已记录的本地 artifact 重新计算 size 与 sha256。
- 恢复后创建一次新备份并完成 verify，然后再允许执行破坏性操作。

## 回滚

如果恢复验证失败：

1. 停止 backend/frontend。
2. 保留 backend logs 与 Postgres logs。
3. 记录失败命令和 request id。
4. 使用已知可用的 dump 重新创建数据库。
5. 在 `/health`、`/ready`、login、audit 与 backup verification 全部通过前，不继续执行破坏性操作。
