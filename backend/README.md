# Backend

Go backend baseline with:
- `gin` HTTP service
- `viper` config loading
- `zap` structured logging
- `gorm` PostgreSQL connection bootstrap

## Run

1. Ensure PostgreSQL is reachable (defaults from `.env.example`).
2. Start service:
   `go run ./cmd/server`

## Endpoints

- `GET /health`
- `GET /api/v1/ping`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me` (JWT protected)
- `GET /api/v1/dashboard/summary` (JWT protected, data source is core-agent via grpc client)
- `GET /api/v1/files/list?path=` (JWT + `files.read`)
- `POST /api/v1/files/read` (JWT + `files.read`)
- `POST /api/v1/files/write` (JWT + `files.write`)
- `POST /api/v1/files/mkdir` (JWT + `files.write`)
- `DELETE /api/v1/files/delete` (JWT + `files.write`)
- `GET /api/v1/services` (JWT + `services.read`)
- `POST /api/v1/services/:name/start` (JWT + `services.manage`)
- `POST /api/v1/services/:name/stop` (JWT + `services.manage`)
- `POST /api/v1/services/:name/restart` (JWT + `services.manage`)
- `GET /api/v1/docker/containers` (JWT + `docker.read`)
- `POST /api/v1/docker/containers/:id/start` (JWT + `docker.manage`)
- `POST /api/v1/docker/containers/:id/stop` (JWT + `docker.manage`)
- `POST /api/v1/docker/containers/:id/restart` (JWT + `docker.manage`)
- `GET /api/v1/docker/images` (JWT + `docker.read`)
- `GET /api/v1/cron` (JWT + `cron.read`)
- `POST /api/v1/cron` (JWT + `cron.manage`)
- `PUT /api/v1/cron/:id` (JWT + `cron.manage`)
- `DELETE /api/v1/cron/:id` (JWT + `cron.manage`)
- `POST /api/v1/cron/:id/enable` (JWT + `cron.manage`)
- `POST /api/v1/cron/:id/disable` (JWT + `cron.manage`)

## Default Admin Bootstrap

When `BOOTSTRAP_ADMIN=true`, backend creates a default admin only when no users exist.
Default credentials come from:
- `DEFAULT_ADMIN_USERNAME`
- `DEFAULT_ADMIN_PASSWORD`
- `DEFAULT_ADMIN_EMAIL`

## Migration Note

Current SQL migrations are in `backend/migrations`.
Recommended execution flow:
1. apply `.up.sql`
2. start backend

## gRPC Client Skeleton

`internal/grpcclient` currently exposes a transport placeholder interface.
After proto code generation is ready, replace placeholder methods with:
- grpc connection setup
- generated `SystemServiceClient` calls
