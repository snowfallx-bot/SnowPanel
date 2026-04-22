# Security Notes

Language: **English** | [简体中文](security.zh-CN.md)

## Security Goals

- Prevent arbitrary command execution.
- Minimize host-impact blast radius for file/process/docker actions.
- Keep clear attribution for sensitive operations.

## Authentication and Authorization

- JWT-based authentication for protected APIs.
- Route-level permission checks (`RequirePermission` middleware).
- Admin bootstrap only when user table is empty.
- In production, backend startup fails fast when `JWT_SECRET` is weak/empty.
- In production with `BOOTSTRAP_ADMIN=true`, weak/missing `DEFAULT_ADMIN_PASSWORD` is rejected.
- In development, if `DEFAULT_ADMIN_PASSWORD` is empty, backend generates a one-time bootstrap password.

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
- File/service/docker/task creation paths are instrumented with audit writes.

## Error Handling

- Unified backend business error model with stable error codes.
- No raw panics leaked to clients through recover middleware.

## Hardening Backlog

- Add first-login forced password rotation for bootstrap admin.
- Add refresh tokens and session revocation.
- Encrypt sensitive settings/secrets at rest.
- Expand rate limiting and lockout policy on login attempts.
