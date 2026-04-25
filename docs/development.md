# Development Guide

Language: **English** | [简体中文](development.zh-CN.md)

## Prerequisites

- Docker + Docker Compose v2
- Go 1.25+
- Rust stable toolchain
- Node.js 22+ (recommended; minimum 20.19.0 for frontend tooling)

## One-Command Local Stack

1. Copy environment template:
   - `cp .env.example .env`
2. Start full stack:
   - `make up`
3. Access services:
   - Frontend: `http://127.0.0.1:5173`
   - Backend health: `http://127.0.0.1:8080/health`
   - Backend readiness: `http://127.0.0.1:8080/ready`
4. Stop stack:
   - `make down`

## Split Local Workflow

When you want local binaries + containerized dependencies:

1. Start PostgreSQL and Redis:
   - `docker compose up -d postgres redis`
   - for local backend binaries, publish dependency ports with:
     `docker compose -f docker-compose.yml -f docker-compose.local.yml up -d postgres redis`
2. Choose one runtime flow:
   - all-local binaries: `make agent`, `make backend`, `make frontend`
   - backend in Docker + host-installed core-agent: `make up-host-agent`

## Useful Commands

- `make logs`: tail compose logs
- `make logs-host-agent`: tail logs for the host-agent compose stack
- `make up-observability`: start the app stack with Prometheus, Alertmanager, OTel Collector, and Jaeger
- `make up-host-agent-observability`: start host-agent mode with the observability stack
- `make down-observability`: stop the compose stack that includes observability services
- `make down-host-agent-observability`: stop the host-agent stack that includes observability services
- `make logs-observability`: tail logs for Prometheus, Alertmanager, OTel Collector, and Jaeger
- `make logs-host-agent-observability`: tail observability logs in host-agent mode
- `make lint`: baseline static checks (`go vet`, `cargo fmt --check`, frontend build)
- `make test`: backend tests + rust tests + frontend test/build flow

## Host-Agent Note

If you are using the host-agent runtime mode, keep using `make up-host-agent` / `make logs-host-agent` for rebuilds and diagnostics. Falling back to plain `docker compose up` or `make up` will drop `docker-compose.host-agent.yml` and point backend back at the containerized `core-agent`.

## Test Coverage Scope

Current test matrix covers:
- backend unit tests across auth, middleware, grpc client, services, and security-sensitive flows
- backend + fake-agent integration-style tests for dashboard, files, services, Docker, and cron contracts
- compose smoke coverage for login, forced password change, refresh rotation, dashboard, files, and logout
- backend integration CI coverage against real backend/core-agent/postgres wiring
- frontend e2e coverage for login, file browsing, and permission-aware navigation

## Coding Conventions

- Handlers should stay focused on binding/response only.
- Business logic belongs to `service` layer.
- DB operations belong to `repository` layer.
- Host-facing operations must stay explicit and validated; no arbitrary command execution.
