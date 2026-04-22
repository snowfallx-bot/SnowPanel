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
- Core-agent gRPC: `50051`
- PostgreSQL: `5432`
- Redis: `6379`

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

## Production Considerations

- Replace default credentials and JWT secret.
- Use persistent backup strategy for Postgres volumes.
- Place backend/frontend behind HTTPS reverse proxy.
- Restrict core-agent exposure to trusted network only.
