# Deployment Guide

Language: **English** | [简体中文](deployment.zh-CN.md)

## Runtime Modes

| Mode | Description | Recommended For |
| --- | --- | --- |
| Compose Mode | `core-agent` runs as a container in the same compose stack. | Local development and demos. |
| Host-Agent (Recommended) | `core-agent` runs as a host systemd service; backend connects over private network gRPC. | Production and real host operations (docker/systemd/cron). |

## Ubuntu 25.10 One-Click Install

For host-agent mode on Ubuntu 25.10, use:

- [One-click installer](../deploy/one-click/ubuntu-25.10/README.md)

## Mode A: Compose Mode

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
   - for production, set `BACKEND_AGENT_AUTH_MODE=token` and use the same `BACKEND_AGENT_SHARED_TOKEN` value as host `CORE_AGENT_SHARED_TOKEN`
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

Run app stack with the observability baseline:

- Compose mode: `make up-observability`
- Host-agent mode: `make up-host-agent-observability`

Observability UIs:

- `http://127.0.0.1:${PROMETHEUS_PORT:-9090}`
- `http://127.0.0.1:${ALERTMANAGER_PORT:-9093}`
- `http://127.0.0.1:${JAEGER_UI_PORT:-16686}`

Stop:

- Compose mode: `make down-observability`
- Host-agent mode: `make down-host-agent-observability`

## Port Defaults (Compose Mode)

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
- backend <-> core-agent auth mode (`BACKEND_AGENT_AUTH_MODE`, `BACKEND_AGENT_SHARED_TOKEN`)
- core-agent auth mode (`CORE_AGENT_AUTH_MODE`, `CORE_AGENT_SHARED_TOKEN`)
- core-agent production safety override (`CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG`, default `false`)
- core-agent operation gates (`CORE_AGENT_ENABLE_FILE_OPS`, `CORE_AGENT_ENABLE_SERVICE_OPS`, `CORE_AGENT_ENABLE_DOCKER_OPS`, `CORE_AGENT_ENABLE_CRON_OPS`)
- durable task worker controls (`TASK_WORKER_ENABLED`, `TASK_WORKER_CONCURRENCY`, `TASK_WORKER_LEASE_DURATION`, `TASK_WORKER_MAX_ATTEMPTS`)
- audit retention/export controls (`AUDIT_RETENTION_DAYS`, `AUDIT_EXPORT_MAX_ROWS`)
- core-agent safe-root and read/write limits
- core-agent metrics endpoint config (`CORE_AGENT_METRICS_ENABLED`, `CORE_AGENT_METRICS_HOST`, `CORE_AGENT_METRICS_PORT`)
- OTEL tracing config (`OTEL_TRACING_ENABLED`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_TRACES_SAMPLER_ARG`)
- PostgreSQL + Redis connection info (`REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`)
- frontend API base URL (`VITE_API_BASE_URL`, prefer empty for same-origin requests)
- frontend Vite proxy target (`VITE_API_PROXY_TARGET`, defaults to backend service in Docker)
- when `APP_ENV=production`, startup fails fast if `JWT_SECRET` is weak/empty
- when `APP_ENV=production` and `BOOTSTRAP_ADMIN=true`, `DEFAULT_ADMIN_PASSWORD` must be strong

## Production Considerations

- Prefer host-agent mode for real host control paths.
- Set `APP_ENV=production` and provide a strong explicit `JWT_SECRET`.
- If bootstrap admin is enabled, provide a strong explicit `DEFAULT_ADMIN_PASSWORD`.
- Enable backend <-> core-agent token auth in production unless the deployment has a stronger private mTLS boundary.
- In production, leave `CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=false`; the agent fails fast when auth is disabled, allowed roots include `/`, service operations have no whitelist, or cron operations rely on default commands.
- Disable unused host-agent operation categories with `CORE_AGENT_ENABLE_*_OPS=false` to reduce blast radius.
- Keep `TASK_WORKER_ENABLED=true` for production so Docker/service restart tasks are executed by the DB-backed durable worker with leases and retries.
- Use `TASK_WORKER_ENABLED=false` only as a temporary local compatibility or rollback mode; it falls back to the legacy in-process goroutine executor.
- Set `AUDIT_RETENTION_DAYS` to the retention period required by your operational policy; default is `180`.
- Keep `AUDIT_EXPORT_MAX_ROWS` bounded for predictable audit export memory/network usage; default is `100000`.
- Use persistent backup strategy for Postgres volumes.
- Place backend/frontend behind HTTPS reverse proxy.
- Restrict core-agent (`50051`) exposure to trusted network only.
- Never expose core-agent gRPC (`50051`) to the public internet.
- Keep core-agent metrics endpoint (`CORE_AGENT_METRICS_HOST:CORE_AGENT_METRICS_PORT`, default `127.0.0.1:9108` in host mode) in loopback or trusted scrape networks.
- If you enable host-agent tracing, point `OTEL_EXPORTER_OTLP_ENDPOINT` at the collector address reachable from host (for local compose observability baseline, `127.0.0.1:4317`).
- Before destructive operations, verify backups exist for Postgres, app metadata, and any host configuration under managed roots.

## Host-Agent Production Checklist

- Bind gRPC to loopback or a private interface when possible; firewall port `50051`.
- Enable agent auth (`CORE_AGENT_AUTH_MODE=token`) and keep backend token config in sync.
- Keep `CORE_AGENT_ALLOWED_ROOTS` narrow and never use `/` in production.
- Keep `CORE_AGENT_SERVICE_WHITELIST` to only services SnowPanel should manage.
- Set `CORE_AGENT_CRON_ALLOWED_COMMANDS` explicitly; do not rely on defaults.
- Treat Docker socket access as host-root-equivalent and disable Docker ops when not needed.
- Keep metrics on loopback or a trusted scrape network.
- Confirm backups before file writes/deletes, service restarts, Docker actions, or cron edits.
