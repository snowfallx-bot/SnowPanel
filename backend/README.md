# Backend

Go API service for SnowPanel. The backend is responsible for:

- HTTP API routing with `gin`
- auth/session flows with JWT + refresh rotation
- RBAC-aware authorization middleware
- PostgreSQL-backed user/audit/task state
- gRPC calls to `core-agent` for host operations
- metrics, request correlation, and tracing hooks

## Run

1. Ensure PostgreSQL is reachable (defaults from `.env.example`).
2. Ensure `core-agent` is reachable via `AGENT_TARGET`.
3. Start service:
   - `go run ./cmd/server`

## Operational Endpoints

- `GET /health`
- `GET /ready`
- `GET /metrics`

## API Surface

Auth:

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/change-password`

Dashboard:

- `GET /api/v1/dashboard/summary`

Files:

- `GET /api/v1/files/list`
- `GET /api/v1/files/download`
- `POST /api/v1/files/upload`
- `POST /api/v1/files/read`
- `POST /api/v1/files/write`
- `POST /api/v1/files/rename`
- `POST /api/v1/files/mkdir`
- `DELETE /api/v1/files/delete`

Services:

- `GET /api/v1/services`
- `POST /api/v1/services/:name/start`
- `POST /api/v1/services/:name/stop`
- `POST /api/v1/services/:name/restart`

Docker:

- `GET /api/v1/docker/containers`
- `POST /api/v1/docker/containers/:id/start`
- `POST /api/v1/docker/containers/:id/stop`
- `POST /api/v1/docker/containers/:id/restart`
- `GET /api/v1/docker/images`

Cron:

- `GET /api/v1/cron`
- `POST /api/v1/cron`
- `PUT /api/v1/cron/:id`
- `DELETE /api/v1/cron/:id`
- `POST /api/v1/cron/:id/enable`
- `POST /api/v1/cron/:id/disable`

Audit / Tasks:

- `GET /api/v1/audit/logs`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/:id`
- `POST /api/v1/tasks/docker/restart`
- `POST /api/v1/tasks/services/restart`
- `POST /api/v1/tasks/:id/cancel`
- `POST /api/v1/tasks/:id/retry`

## Auth Bootstrap

When `BOOTSTRAP_ADMIN=true`, backend creates the first admin only when no users exist.
Bootstrap values come from:

- `DEFAULT_ADMIN_USERNAME`
- `DEFAULT_ADMIN_PASSWORD`
- `DEFAULT_ADMIN_EMAIL`

In development, an empty `DEFAULT_ADMIN_PASSWORD` generates a one-time password.
In production, startup fails if bootstrap password or `JWT_SECRET` is weak.

## Data And Migrations

Schema migrations live in `backend/migrations`.
Recommended flow:

1. Apply `.up.sql`
2. Start backend

## Observability Notes

- Prometheus metrics are exposed at `/metrics`.
- `X-Request-ID` is propagated from HTTP to gRPC metadata.
- Access logs include `request_id`, `trace_id`, and `span_id`.
- OTEL tracing can be enabled with:
  - `OTEL_TRACING_ENABLED=true`
  - `OTEL_EXPORTER_OTLP_ENDPOINT=<collector-host>:4317`

See repository docs for full deployment and observability guidance:

- `docs/deployment.md`
- `docs/observability.md`
