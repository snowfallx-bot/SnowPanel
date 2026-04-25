# 开发指南

语言: [English](development.md) | **简体中文**

## 前置要求

- Docker + Docker Compose v2
- Go 1.25+
- Rust stable toolchain
- Node.js 22+（推荐；前端工具链最低 20.19.0）

## 一键本地环境

1. 复制环境模板：
   - `cp .env.example .env`
2. 启动完整服务栈：
   - `make up`
3. 访问服务：
   - Frontend：`http://127.0.0.1:5173`
   - Backend 健康检查：`http://127.0.0.1:8080/health`
   - Backend 就绪检查：`http://127.0.0.1:8080/ready`
4. 停止服务栈：
   - `make down`

## 分离式本地工作流

当你希望“依赖容器化 + 服务本地二进制运行”时：

1. 启动 PostgreSQL 和 Redis：
   - `docker compose up -d postgres redis`
   - 若 backend 以本机二进制运行，需要按需暴露依赖端口：
     `docker compose -f docker-compose.yml -f docker-compose.local.yml up -d postgres redis`
2. 选择一种运行方式：
   - 全本地二进制：`make agent`、`make backend`、`make frontend`
   - backend 容器 + 宿主机 core-agent：`make up-host-agent`

## 常用命令

- `make logs`：查看 compose 日志
- `make logs-host-agent`：查看宿主机 Agent 覆盖模式的 compose 日志
- `make up-observability`：以 Prometheus、Alertmanager、OTel Collector、Jaeger 一起启动应用栈
- `make up-host-agent-observability`：以宿主机 Agent 模式启动应用栈并附带可观测性组件
- `make down-observability`：停止附带可观测性组件的 compose 栈
- `make down-host-agent-observability`：停止宿主机 Agent + 可观测性模式的 compose 栈
- `make logs-observability`：查看 Prometheus、Alertmanager、OTel Collector、Jaeger 日志
- `make logs-host-agent-observability`：查看宿主机 Agent + 可观测性模式下的 observability 日志
- `pwsh -File ./scripts/observability/trace-smoke.ps1 -AccessToken "<access_token>"`：触发一次 core-agent 请求，并校验 Jaeger 中 backend/core-agent spans 是否串联
- `make lint`：基础静态检查（`go vet`、`cargo fmt --check`、frontend build）
- `make test`：backend 测试 + rust 测试 + frontend test/build 流程

## 宿主机 Agent 模式提示

如果你使用的是宿主机 Agent 运行模式，后续重建和排障请持续使用 `make up-host-agent` / `make logs-host-agent`。如果误用普通 `docker compose up` 或 `make up`，会丢掉 `docker-compose.host-agent.yml` 覆盖，backend 也会重新指回容器内的 `core-agent`。

## 测试覆盖范围

当前测试矩阵已覆盖：
- backend unit tests：auth、middleware、grpc client、service 层与安全敏感路径
- backend + fake-agent integration-style tests：dashboard、files、services、Docker、cron 契约
- compose smoke：登录、首次强制改密、refresh rotation、dashboard、files、logout 主链路
- backend integration CI：真实 backend/core-agent/postgres 组合链路
- frontend e2e：登录、文件浏览、权限感知导航

## 编码约定

- handler 仅负责参数绑定与响应组装。
- 业务逻辑应放在 `service` 层。
- 数据库操作应放在 `repository` 层。
- 面向主机的操作必须显式且可校验，不允许任意命令执行。
