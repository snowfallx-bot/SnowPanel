# 部署指南

语言: [English](deployment.md) | **简体中文**

## Docker Compose 部署（原型）

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
4. 停止服务：
   - `docker compose down`

## 默认端口

- Frontend：`5173`
- Backend：`8080`
- Core-agent gRPC：`50051`
- PostgreSQL：`5432`
- Redis：`6379`

## 数据库初始化

PostgreSQL 首次初始化时，会加载以下 schema SQL：
- `backend/migrations/0001_init_schema.up.sql`

并挂载到：
- `/docker-entrypoint-initdb.d/0001_init_schema.sql`

## 环境变量说明

`.env` 中关键配置包括：
- backend host/port/JWT/管理员初始化变量
- core-agent 安全根目录与读写大小限制
- PostgreSQL + Redis 连接参数
- frontend API 基地址（`VITE_API_BASE_URL`）

## 生产环境建议

- 替换默认凭据与 JWT 密钥。
- 为 Postgres 数据卷配置持久化备份策略。
- 在 backend/frontend 前加 HTTPS 反向代理。
- 仅在可信网络暴露 core-agent。
