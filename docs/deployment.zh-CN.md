# 部署指南

语言: [English](deployment.md) | **简体中文**

## 运行模式

| 模式 | 描述 | 推荐场景 |
| --- | --- | --- |
| Compose 原型模式 | `core-agent` 与其他服务一起跑在 compose 内。 | 本地开发与演示环境。 |
| 宿主机 Agent 模式（推荐） | `core-agent` 以宿主机 systemd 服务运行，backend 通过内网 gRPC 连接。 | 生产环境与真实宿主机运维场景（docker/systemd/cron）。 |

## Ubuntu 25.10 一键安装

若目标环境是 Ubuntu 25.10 且使用“宿主机 Agent 模式”，可直接使用：

- [一键安装脚本说明](../deploy/one-click/ubuntu-25.10/README.zh-CN.md)

## 模式 A：Compose 原型模式

项目提供了面向开发阶段的 compose 服务栈，包括：
- `postgres`
- `redis`
- `core-agent`
- `backend`
- `frontend`

## 部署步骤

1. 准备环境变量：
   - `cp .env.example .env`
2. 启动服务：
   - `docker compose up -d --build`
3. 验证状态：
   - `docker compose ps`
   - `curl http://127.0.0.1:8080/health`
   - `curl http://127.0.0.1:8080/ready`
4. 停止服务：
   - `docker compose down`

## 模式 B：宿主机 Agent 模式（推荐）

1. 先在宿主机部署 `core-agent`：
   - [systemd 部署模板](../deploy/core-agent/systemd/README.zh-CN.md)
2. 准备应用环境变量：
   - `cp .env.example .env`
   - 将 `AGENT_TARGET` 指向宿主机可访问地址（例如 backend 在 Docker 中时可用 `host.docker.internal:50051`）
3. 使用宿主机 agent 覆盖配置启动 backend/frontend 与依赖：
   - `make up-host-agent`
4. 验证：
   - `curl http://127.0.0.1:8080/health`
   - `curl http://127.0.0.1:8080/ready`

后续如果需要在宿主机 Agent 模式下重建或看日志，请持续使用：

- `make up-host-agent`
- `make logs-host-agent`

不要再退回普通 `docker compose up` / `make up`，否则 backend 会丢失 host-agent 覆盖，重新连回已禁用的容器版 `core-agent`。

## 可选：可观测性基线

以 Prometheus 基线一起启动应用栈：

- Compose 模式：`make up-observability`
- 宿主机 Agent 模式：`make up-host-agent-observability`

Prometheus UI：

- `http://127.0.0.1:${PROMETHEUS_PORT:-9090}`
- `http://127.0.0.1:${ALERTMANAGER_PORT:-9093}`
- `http://127.0.0.1:${JAEGER_UI_PORT:-16686}`

停止：

- Compose 模式：`make down-observability`
- 宿主机 Agent 模式：`make down-host-agent-observability`

## 默认端口（Compose 原型）

- Frontend：`5173`
- Backend：`8080`
- Core-agent gRPC：默认仅容器内部可见（Compose 网络内 `50051`，默认不暴露到宿主机）
- Core-agent metrics：默认仅容器内部可见（Compose 网络内 `9108`，默认不暴露到宿主机）
- PostgreSQL：默认仅容器内部可见（Compose 网络内 `5432`，默认不暴露到宿主机）
- Redis：默认仅容器内部可见（Compose 网络内 `6379`，默认不暴露到宿主机）

## 数据库初始化

PostgreSQL 首次初始化时，会加载以下 schema SQL：
- `backend/migrations/0001_init_schema.up.sql`

并挂载到：
- `/docker-entrypoint-initdb.d/0001_init_schema.sql`

## 环境变量说明

`.env` 中关键配置包括：
- backend host/port/JWT/管理员初始化变量
- 令牌有效期配置（`JWT_EXPIRE`、`JWT_REFRESH_EXPIRE`）
- 登录防爆破模式与阈值（`LOGIN_ATTEMPT_STORE`、`LOGIN_ATTEMPT_REDIS_PREFIX`、`LOGIN_*`）
- core-agent 安全根目录与读写大小限制
- core-agent 指标端点配置（`CORE_AGENT_METRICS_ENABLED`、`CORE_AGENT_METRICS_HOST`、`CORE_AGENT_METRICS_PORT`）
- OTEL tracing 配置（`OTEL_TRACING_ENABLED`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_TRACES_SAMPLER_ARG`）
- PostgreSQL + Redis 连接参数（`REDIS_HOST`、`REDIS_PORT`、`REDIS_PASSWORD`、`REDIS_DB`）
- frontend API 基地址（`VITE_API_BASE_URL`，推荐留空并走同源请求）
- frontend 的 Vite 代理目标（`VITE_API_PROXY_TARGET`，Docker 下默认指向 backend 服务）
- 当 `APP_ENV=production` 时，若 `JWT_SECRET` 为空或过弱，启动会 fail fast
- 当 `APP_ENV=production` 且 `BOOTSTRAP_ADMIN=true` 时，`DEFAULT_ADMIN_PASSWORD` 必须为强密码

## 生产环境建议

- 真实宿主机控制链路优先使用“宿主机 Agent 模式”。
- 设置 `APP_ENV=production` 并显式提供强 `JWT_SECRET`。
- 若启用管理员初始化（`BOOTSTRAP_ADMIN=true`），显式提供强 `DEFAULT_ADMIN_PASSWORD`。
- 为 Postgres 数据卷配置持久化备份策略。
- 在 backend/frontend 前加 HTTPS 反向代理。
- 仅在可信网络暴露 core-agent（`50051`）。
- 将 core-agent metrics 端点（宿主机模式默认 `127.0.0.1:9108`）限制在本地或可信采集网络。
- 若宿主机 Agent 模式启用 tracing，请将 `OTEL_EXPORTER_OTLP_ENDPOINT` 指向宿主机可访问的 collector 地址（本仓库 compose 可观测性基线下可用 `127.0.0.1:4317`）。
