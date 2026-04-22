# Architecture

Language: **English** | [简体中文](architecture.zh-CN.md)

## System Components

SnowPanel is a monorepo with three runtime services:
- `frontend` (`React + TypeScript + Vite`): operator UI (login, dashboard, files, services, docker, cron, audit, tasks).
- `backend` (`Go + Gin + GORM`): authentication, RBAC, API orchestration, audit logging, async task lifecycle.
- `core-agent` (`Rust + tonic`): controlled host operations exposed over gRPC.

## Runtime Topology

1. Browser sends HTTP requests to backend REST APIs.
2. Backend validates JWT + permission middleware before entering handlers.
3. Backend services call core-agent over gRPC for host operations.
4. Backend persists business/audit/task data to PostgreSQL.
5. Redis is reserved for cache/transient use in later iterations.

## Layering Conventions

Backend keeps handlers thin and uses:
- `dto`: request/response structures.
- `service`: business orchestration.
- `repository`: database access.
- `middleware`: request ID, recovery, access log, JWT, permission check.

Core-agent keeps system capabilities in explicit modules:
- `file`: secure file access with path validation.
- `process`: service management (systemctl).
- `docker`: docker container/image operations.
- `cron`: cron task operations.
- `service/system_info`: host overview and realtime resources.

## Data Model Highlights

Main PostgreSQL tables currently used:
- `users`: operator credentials and status.
- `audit_logs`: immutable operation records.
- `tasks` and `task_logs`: async task status/progress/log stream.

Additional schema is prepared for future modules (`websites`, `plugins`, `backups`, etc.).

## Security Boundaries

- No arbitrary command execution API is exposed.
- File operations go through safe-root and dangerous-path checks.
- Operation permissions are enforced at route level.
- Critical operations write audit records with user/IP/module/action/result.
