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
  - response: access token + user profile + permission list
- `GET /auth/me` (protected)
  - response: current user profile

## Dashboard

- `GET /dashboard/summary` (protected, `dashboard.read`)
  - source of truth: core-agent gRPC system overview

## File Management

- `GET /files/list?path=/abs/path` (`files.read`)
- `POST /files/read` (`files.read`)
- `POST /files/write` (`files.write`)
- `POST /files/mkdir` (`files.write`)
- `DELETE /files/delete` (`files.write`)

All file paths are validated by the agent safe-root policy.

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
- `POST /tasks/demo` (`tasks.manage`)
  - creates demo mock-backup task with background progress updates and task logs
