# 开发指南

语言: [English](development.md) | **简体中文**

## 前置要求

- Docker + Docker Compose v2
- Go 1.25+
- Rust stable toolchain
- Node.js 22+

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
2. 运行 core-agent：
   - `make agent`
3. 运行 backend：
   - `make backend`
4. 运行 frontend：
   - `make frontend`

## 常用命令

- `make logs`：查看 compose 日志
- `make lint`：基础静态检查（`go vet`、`cargo fmt --check`、frontend build）
- `make test`：backend 测试 + rust 测试 + frontend test/build 流程

## 测试覆盖范围

当前最小测试集重点覆盖：
- backend auth service 与 auth/permission middleware 行为。
- core-agent 路径校验器与系统信息服务基本正确性。
- frontend 登录页渲染基线。

## 编码约定

- handler 仅负责参数绑定与响应组装。
- 业务逻辑应放在 `service` 层。
- 数据库操作应放在 `repository` 层。
- 面向主机的操作必须显式且可校验，不允许任意命令执行。
