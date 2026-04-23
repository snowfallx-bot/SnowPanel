# API Design

Language: **English** | [简体中文](api-design.zh-CN.md)

## Base Information

- Base path: `/api/v1`
- Transport: JSON over HTTP
- Auth: `Authorization: Bearer <token>` for protected routes
- Response envelope:

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

`code != 0` indicates business failure.

## Authentication

- `POST /auth/login`
  - request: `{ "username": "...", "password": "..." }`
  - response: access token + refresh token + user profile + permission list
  - note: bootstrap admin first login may return `user.must_change_password = true`
  - security note: too many failed attempts for the same `username + client IP` are temporarily locked with `429`
- `POST /auth/refresh`
  - request: `{ "refresh_token": "..." }`
  - response: rotated access token + rotated refresh token + latest user profile
- `POST /auth/logout` (protected)
  - behavior: revokes current logical session by rotating session timestamp
- `GET /auth/me` (protected)
  - response: current user profile
- `POST /auth/change-password` (protected)
  - request: `{ "current_password": "...", "new_password": "..." }`
  - response: refreshed access token + refreshed refresh token + updated user profile

## Dashboard

- `GET /dashboard/summary` (protected, `dashboard.read`)
  - source of truth: core-agent gRPC system overview

## File Management

- `GET /files/list?path=/abs/path` (`files.read`)
- `GET /files/download?path=/abs/path` (`files.read`)
- `POST /files/read` (`files.read`)
- `POST /files/write` (`files.write`)
- `POST /files/rename` (`files.write`)
- `POST /files/mkdir` (`files.write`)
- `DELETE /files/delete` (`files.write`)

All file paths are validated by the agent safe-root policy.

Current behavior notes:
- File read/write APIs are text-oriented (`utf-8`) and return `truncated` when max preview bytes are exceeded.
- `GET /files/download` currently proxies text reads (UTF-8, max `8MB`) and returns a `text/plain` attachment.
- Binary or non-UTF-8 files are shown with a clear hint and inline editing is disabled.
- Preview size is selectable (`256KB` to `8MB`); offset/chunk pagination is not yet exposed as a dedicated API.
- File-related error codes currently used by the core-agent:
  - `4001`: unsafe path
  - `4002`: path not found
  - `4003`: text file required (binary/non-UTF-8)
  - `4004`: file too large
  - `4005`: I/O error
  - `4006`: unsupported encoding
  - `4007`: dangerous path

## Service Management

- `GET /services` (`services.read`)
- `POST /services/:name/start` (`services.manage`)
- `POST /services/:name/stop` (`services.manage`)
- `POST /services/:name/restart` (`services.manage`)

## Docker Management

- `GET /docker/containers` (`docker.read`)
- `POST /docker/containers/:id/start` (`docker.manage`)
- `POST /docker/containers/:id/stop` (`docker.manage`)
- `POST /docker/containers/:id/restart` (`docker.manage`)
- `GET /docker/images` (`docker.read`)

## Cron Management

- `GET /cron` (`cron.read`)
- `POST /cron` (`cron.manage`)
- `PUT /cron/:id` (`cron.manage`)
- `DELETE /cron/:id` (`cron.manage`)
- `POST /cron/:id/enable` (`cron.manage`)
- `POST /cron/:id/disable` (`cron.manage`)

Security constraints:
- `command` is treated as a command template key, not arbitrary shell text.
- Only commands configured in `CORE_AGENT_CRON_ALLOWED_COMMANDS` are accepted.
- Shell metacharacters (`|`, `&`, `;`, `>`, `<`, `` ` ``, `$`, etc.) are rejected.

## Audit Log

- `GET /audit/logs` (`audit.read`)
  - query: `page`, `size`, optional `module`, `action`

## Async Tasks

- `GET /tasks` (`tasks.read`)
  - query: `page`, `size`
- `GET /tasks/:id` (`tasks.read`)
- `POST /tasks/docker/restart` (`tasks.manage`)
  - body: `{ "container_id": "..." }`
  - queues a real docker restart operation as background task
- `POST /tasks/services/restart` (`tasks.manage`)
  - body: `{ "service_name": "..." }`
  - queues a real system service restart operation as background task
- `POST /tasks/:id/cancel` (`tasks.manage`)
  - cancels a pending/running task
- `POST /tasks/:id/retry` (`tasks.manage`)
  - retries a failed/canceled task using its original payload
