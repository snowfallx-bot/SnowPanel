# Frontend Token Storage Decision

Language: **English** | [简体中文](security-token-storage-decision.zh-CN.md)

Date: 2026-05-10

## Decision

SnowPanel will keep access and refresh tokens in the persisted frontend auth store for the current P3 hardening phase.

This is an accepted interim risk, not the final target. A future migration to backend-issued httpOnly secure cookies plus CSRF protection remains the preferred browser-only posture once the surrounding deployment assumptions are ready.

## Current Risk

- Any successful XSS could read persisted bearer and refresh tokens from browser storage.
- Browser extensions or compromised local profiles could access persisted tokens.
- Refresh tokens increase the value of a browser-storage compromise because they can mint a new access token until backend session controls revoke them.

## Current Compensating Controls

- Backend validates token session state against the DB user status and `last_login_at`.
- Re-login, password change, logout, and disabled-user handling revoke old logical sessions.
- `/auth/refresh` rotates both access and refresh tokens.
- Protected frontend routes call `/auth/me` to validate session state.
- `401` responses clear local auth state and redirect the operator to login.

## Why Not Move Now

- SnowPanel still supports simple Bearer-token API client flows.
- The current local and reverse-proxy deployment paths are simpler without cookie domain, SameSite, and trusted-proxy coupling.
- Moving only storage without CSRF, cookie domain rules, secure proxy headers, and regression tests would create a partial security migration with unclear operator behavior.

## Migration Preconditions

Before migrating browser sessions to httpOnly cookies, implement and test:

- Backend-issued `Secure`, `HttpOnly`, `SameSite` cookies for access and refresh state.
- CSRF protection for unsafe methods.
- Trusted reverse-proxy and domain configuration.
- Browser regression tests covering login, refresh, logout, password change, and `401` recovery.
- API-client compatibility plan for non-browser automation.

## Future Plan

1. Add cookie session support behind a feature flag.
2. Add CSRF middleware and frontend token submission for unsafe methods.
3. Run Bearer and cookie modes side-by-side in tests.
4. Document production cookie/domain/proxy requirements.
5. Switch browser deployments to cookie mode after regression coverage is stable.
