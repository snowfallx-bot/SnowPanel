# API 设计

语言: [English](api-design.md) | **简体中文**

## 基础信息

- 基础路径：`/api/v1`
- 传输协议：HTTP + JSON
- 鉴权方式：受保护路由使用 `Authorization: Bearer <token>`
- 响应包裹结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

`code != 0` 表示业务失败。

## 认证

- `POST /auth/login`
  - 请求：`{ "username": "...", "password": "..." }`
  - 响应：访问令牌 + 用户信息 + 权限列表
- `GET /auth/me`（受保护）
  - 响应：当前用户信息

## 仪表盘

- `GET /dashboard/summary`（受保护，`dashboard.read`）
  - 数据来源：core-agent gRPC 系统概览

## 文件管理

- `GET /files/list?path=/abs/path`（`files.read`）
- `POST /files/read`（`files.read`）
- `POST /files/write`（`files.write`）
- `POST /files/mkdir`（`files.write`）
- `DELETE /files/delete`（`files.write`）

所有文件路径都由 agent 的安全根目录策略进行校验。

## 服务管理

- `GET /services`（`services.read`）
- `POST /services/:name/start`（`services.manage`）
- `POST /services/:name/stop`（`services.manage`）
- `POST /services/:name/restart`（`services.manage`）

## Docker 管理

- `GET /docker/containers`（`docker.read`）
- `POST /docker/containers/:id/start`（`docker.manage`）
- `POST /docker/containers/:id/stop`（`docker.manage`）
- `POST /docker/containers/:id/restart`（`docker.manage`）
- `GET /docker/images`（`docker.read`）

## Cron 管理

- `GET /cron`（`cron.read`）
- `POST /cron`（`cron.manage`）
- `PUT /cron/:id`（`cron.manage`）
- `DELETE /cron/:id`（`cron.manage`）
- `POST /cron/:id/enable`（`cron.manage`）
- `POST /cron/:id/disable`（`cron.manage`）

## 审计日志

- `GET /audit/logs`（`audit.read`）
  - 查询参数：`page`、`size`，可选 `module`、`action`

## 异步任务

- `GET /tasks`（`tasks.read`）
  - 查询参数：`page`、`size`
- `GET /tasks/:id`（`tasks.read`）
- `POST /tasks/demo`（`tasks.manage`）
  - 创建示例 mock-backup 任务，并在后台更新进度和任务日志
