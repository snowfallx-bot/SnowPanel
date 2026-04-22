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
