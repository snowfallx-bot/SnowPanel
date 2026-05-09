# 架构说明

语言: [English](architecture.md) | **简体中文**

## 系统组件

SnowPanel 是一个由三类运行时服务组成的 monorepo：
- `frontend`（`React + TypeScript + Vite`）：运维操作 UI（登录、仪表盘、文件、服务、Docker、Cron、审计、任务）。
- `backend`（`Go + Gin + GORM`）：鉴权、RBAC、API 编排、审计日志、异步任务生命周期。
- `core-agent`（`Rust + tonic`）：通过 gRPC 暴露受控的主机操作能力。

## 运行拓扑

1. 浏览器向后端 REST API 发起 HTTP 请求。
2. 后端在进入 handler 之前执行 JWT 与权限中间件校验。
3. 后端服务通过 gRPC 调用 core-agent 执行主机操作。
4. 后端将业务/审计/任务数据写入 PostgreSQL。
5. Redis 当前已可用于共享登录失败限流状态，同时仍预留给其他临时数据场景。

## 分层约定

后端保持 handler 轻量，并采用以下分层：
- `dto`：请求/响应结构。
- `service`：业务编排。
- `repository`：数据库访问。
- `middleware`：请求 ID、recover、访问日志、JWT、权限校验。

core-agent 将系统能力按模块显式拆分：
- `file`：带路径校验的安全文件访问。
- `process`：服务管理（systemctl）。
- `docker`：Docker 容器/镜像操作。
- `cron`：Cron 任务操作。
- `service/system_info`：主机概览与实时资源数据。

## 数据模型重点

当前主要使用的 PostgreSQL 表：
- `users`：操作员凭据与状态。
- `audit_logs`：不可变操作记录。
- `tasks` 和 `task_logs`：异步任务状态/进度/日志流。

额外 schema 已为未来模块预留（`websites`、`plugins`、`backups` 等）。

## 安全边界

- 不暴露任意命令执行 API。
- 文件操作必须通过安全根目录与危险路径检查。
- 操作权限在路由级别强制执行。
- 关键操作写入审计记录，包含用户/IP/模块/动作/结果。
