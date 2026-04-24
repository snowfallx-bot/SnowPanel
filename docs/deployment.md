# Deployment Guide

Language: **English** | [简体中文](deployment.zh-CN.md)

## Runtime Modes

| Mode | Description | Recommended For |
| --- | --- | --- |
| Compose Prototype | `core-agent` runs as a container in the same compose stack. | Local development and demos. |
| Host-Agent (Recommended) | `core-agent` runs as a host systemd service; backend connects over private network gRPC. | Production and real host operations (docker/systemd/cron). |

## Ubuntu 25.10 One-Click Install

For host-agent mode on Ubuntu 25.10, use:

- [One-click installer](../deploy/one-click/ubuntu-25.10/README.md)

## Mode A: Compose Prototype

This project ships with a development-oriented compose stack including:
- `postgres`
- `redis`
- `core-agent`
- `backend`
- `frontend`

## Steps

1. Prepare environment:
   - `cp .env.example .env`
2. Start services:
   - `docker compose up -d --build`
3. Verify:
   - `docker compose ps`
   - `curl http://127.0.0.1:8080/health`
   - `curl http://127.0.0.1:8080/ready`
4. Stop:
   - `docker compose down`

## Mode B: Host-Agent (Recommended)

1. Prepare host `core-agent` service from deployment assets:
   - [Systemd deployment template](../deploy/core-agent/systemd/README.md)
2. Prepare app environment:
   - `cp .env.example .env`
   - set `AGENT_TARGET` to host-accessible address (for example `host.docker.internal:50051` when backend runs in Docker)
3. Start backend/frontend + dependencies with host-agent override:
   - `make up-host-agent`
4. Verify:
   - `curl http://127.0.0.1:8080/health`
   - `curl http://127.0.0.1:8080/ready`

For later rebuilds and log inspection in host-agent mode, keep using:

- `make up-host-agent`
- `make logs-host-agent`

Do not fall back to plain `docker compose up` / `make up`, or backend will lose the host-agent override and reconnect to the disabled containerized `core-agent`.

## Optional: Observability Baseline

Run app stack with Prometheus baseline:

- Compose mode: `make up-observability`
- Host-agent mode: `make up-host-agent-observability`

Prometheus UI:

- `http://127.0.0.1:${PROMETHEUS_PORT:-9090}`
- `http://127.0.0.1:${ALERTMANAGER_PORT:-9093}`

Stop:

- Compose mode: `make down-observability`
- Host-agent mode: `make down-host-agent-observability`

## Port Defaults (Prototype Compose)

- Frontend: `5173`
- Backend: `8080`
- Core-agent gRPC: internal-only (`50051` in Compose network, not exposed on host by default)
- Core-agent metrics: internal-only (`9108` in Compose network, not exposed on host by default)
- PostgreSQL: internal-only (`5432` in Compose network, not exposed on host by default)
- Redis: internal-only (`6379` in Compose network, not exposed on host by default)

## Database Initialization

On first PostgreSQL initialization, schema SQL is loaded from:
- `backend/migrations/0001_init_schema.up.sql`

It is mounted to:
- `/docker-entrypoint-initdb.d/0001_init_schema.sql`

## Environment Notes

Key settings in `.env`:
- backend host/port/JWT/admin bootstrap variables
- token lifetimes (`JWT_EXPIRE`, `JWT_REFRESH_EXPIRE`)
- login attempt limiter mode and thresholds (`LOGIN_ATTEMPT_STORE`, `LOGIN_ATTEMPT_REDIS_PREFIX`, `LOGIN_*`)
- core-agent safe-root and read/write limits
- core-agent metrics endpoint config (`CORE_AGENT_METRICS_ENABLED`, `CORE_AGENT_METRICS_HOST`, `CORE_AGENT_METRICS_PORT`)
- PostgreSQL + Redis connection info (`REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`)
- frontend API base URL (`VITE_API_BASE_URL`, prefer empty for same-origin requests)
- frontend Vite proxy target (`VITE_API_PROXY_TARGET`, defaults to backend service in Docker)
- when `APP_ENV=production`, startup fails fast if `JWT_SECRET` is weak/empty
- when `APP_ENV=production` and `BOOTSTRAP_ADMIN=true`, `DEFAULT_ADMIN_PASSWORD` must be strong

## Production Considerations

- Prefer host-agent mode for real host control paths.
- Set `APP_ENV=production` and provide a strong explicit `JWT_SECRET`.
- If bootstrap admin is enabled, provide a strong explicit `DEFAULT_ADMIN_PASSWORD`.
- Use persistent backup strategy for Postgres volumes.
- Place backend/frontend behind HTTPS reverse proxy.
- Restrict core-agent (`50051`) exposure to trusted network only.
- Keep core-agent metrics endpoint (`CORE_AGENT_METRICS_HOST:CORE_AGENT_METRICS_PORT`, default `127.0.0.1:9108` in host mode) in loopback or trusted scrape networks.
