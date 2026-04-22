# Deployment Guide

Language: **English** | [简体中文](deployment.zh-CN.md)

## Docker Compose Deployment (Prototype)

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
4. Stop:
   - `docker compose down`

## Port Defaults

- Frontend: `5173`
- Backend: `8080`
- Core-agent gRPC: internal-only (`50051` in Compose network, not exposed on host by default)
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
- core-agent safe-root and read/write limits
- PostgreSQL + Redis connection info
- frontend API base URL (`VITE_API_BASE_URL`)
- when `APP_ENV=production`, startup fails fast if `JWT_SECRET` is weak/empty
- when `APP_ENV=production` and `BOOTSTRAP_ADMIN=true`, `DEFAULT_ADMIN_PASSWORD` must be strong

## Production Considerations

- Set `APP_ENV=production` and provide a strong explicit `JWT_SECRET`.
- If bootstrap admin is enabled, provide a strong explicit `DEFAULT_ADMIN_PASSWORD`.
- Use persistent backup strategy for Postgres volumes.
- Place backend/frontend behind HTTPS reverse proxy.
- Restrict core-agent exposure to trusted network only.
