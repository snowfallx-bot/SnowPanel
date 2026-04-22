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
- `POST /files/rename`（`files.write`）
- `POST /files/mkdir`（`files.write`）
- `DELETE /files/delete`（`files.write`）

所有文件路径都由 agent 的安全根目录策略进行校验。

当前行为说明：
- 文件读写 API 为文本导向（`utf-8`），超出最大预览字节数时返回 `truncated`。
- 前端的上传/下载能力当前通过复用读写接口实现，面向 UTF-8 文本文件。
- 二进制或非 UTF-8 文件会给出明确提示，并禁用内联编辑。
- 预览大小可选（`256KB` 到 `8MB`）；尚未提供独立的 offset/chunk 分页读取 API。
- 当前 core-agent 文件相关错误码语义：
  - `4001`：unsafe path（路径越界）
  - `4002`：path not found（路径不存在）
  - `4003`：text file required（二进制或非 UTF-8）
  - `4004`：file too large（文件过大）
  - `4005`：I/O error（文件系统读写异常）
  - `4006`：unsupported encoding（编码不支持）
  - `4007`：dangerous path（危险路径）

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

安全约束：
- `command` 按“命令模板标识”处理，不再接受任意 shell 文本。
- 仅允许 `CORE_AGENT_CRON_ALLOWED_COMMANDS` 配置中的命令。
- 会拒绝 shell 元字符（`|`、`&`、`;`、`>`、`<`、`` ` ``、`$` 等）。

## 审计日志

- `GET /audit/logs`（`audit.read`）
  - 查询参数：`page`、`size`，可选 `module`、`action`

## 异步任务

- `GET /tasks`（`tasks.read`）
  - 查询参数：`page`、`size`
- `GET /tasks/:id`（`tasks.read`）
- `POST /tasks/docker/restart`（`tasks.manage`）
  - 请求体：`{ "container_id": "..." }`
  - 将真实 Docker 重启动作加入后台任务队列
- `POST /tasks/services/restart`（`tasks.manage`）
  - 请求体：`{ "service_name": "..." }`
  - 将真实 system service 重启动作加入后台任务队列
- `POST /tasks/:id/cancel`（`tasks.manage`）
  - 取消 `pending/running` 任务
- `POST /tasks/:id/retry`（`tasks.manage`）
  - 基于原始 payload 重试 `failed/canceled` 任务
