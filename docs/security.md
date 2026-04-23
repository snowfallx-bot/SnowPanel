# Security Notes

Language: **English** | [简体中文](security.zh-CN.md)

## Security Goals

- Prevent arbitrary command execution.
- Minimize host-impact blast radius for file/process/docker actions.
- Keep clear attribution for sensitive operations.

## Authentication and Authorization

- JWT-based authentication for protected APIs.
- Route-level permission checks (`RequirePermission` middleware).
- Authorization claims are derived from DB-backed `roles`, `permissions`, and mapping tables.
- No hardcoded username bypass in permission checks.
- Admin bootstrap only when user table is empty.
- Frontend validates session via `getMe()` on protected-route entry.
- Frontend navigation is permission-aware and automatically redirects on `401`.
- In production, backend startup fails fast when `JWT_SECRET` is weak/empty.
- In production with `BOOTSTRAP_ADMIN=true`, weak/missing `DEFAULT_ADMIN_PASSWORD` is rejected.
- In development, if `DEFAULT_ADMIN_PASSWORD` is empty, backend generates a one-time bootstrap password.
- Bootstrap admin first login is marked as `must_change_password`.
- When `must_change_password=true`, backend only allows `/api/v1/auth/me` and `/api/v1/auth/change-password`.
- Password change returns a refreshed token with `must_change_password=false`.
- Backend validates token session state against DB user status and `last_login_at`.
- Old tokens are revoked after re-login/password change, and disabled users lose active sessions.
- Backend also validates a token RBAC checksum against current DB roles/permissions, so role/permission changes force re-authentication.
- Access/refresh token pair is supported; `/auth/refresh` rotates both tokens and advances session timestamp.
- `/auth/logout` revokes current logical session by rotating session timestamp.
- Login endpoint has brute-force protection keyed by `username + client IP`.
- Default mode is in-memory (`LOGIN_ATTEMPT_STORE=memory`); distributed mode can be enabled with Redis (`LOGIN_ATTEMPT_STORE=redis` + shared `REDIS_*` config).
- Repeated failures within `LOGIN_FAILURE_WINDOW` trigger temporary lockout (`429`) for `LOGIN_LOCK_DURATION`.

## File Safety

- Agent requires absolute paths.
- Safe-root policy restricts path access to allowed roots.
- Dangerous delete/write/mkdir targets (`/`, `/etc`, `/usr`, etc.) are blocked.
- Read/write size limits are configurable via env.

## Operational Safety

- Service operations are explicit (`start/stop/restart`) and name-validated.
- Docker operations are explicit (`start/stop/restart/list`) with no shell passthrough.
- Cron operations use structured model + validation flow.
- Cron command scheduling is restricted to allowlisted command templates
  (`CORE_AGENT_CRON_ALLOWED_COMMANDS`) and blocks shell metacharacters.

## Auditability

- Audit records include user id, username, IP, module, action, target, request summary, and result.
- File/service/docker/cron/task operation paths are instrumented with audit writes.

## Error Handling

- Unified backend business error model with stable error codes.
- No raw panics leaked to clients through recover middleware.

## Hardening Backlog

- Encrypt sensitive settings/secrets at rest.
- For multi-region deployments, evaluate cross-region shared rate-limit state and failover behavior.
