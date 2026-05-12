# SnowPanel Restore Drill

This drill defines the minimum restore path for the P3 backup foundation. It is intentionally scoped to SnowPanel control-plane recovery, not arbitrary host or Docker-volume recovery.

## Backup Scope

Included in P3:

- Postgres database dump containing SnowPanel metadata.
- SnowPanel app metadata tracked in the database.
- Observability configuration snapshot under `deploy/observability`.
- Core-agent configuration templates under `deploy/core-agent`.
- Backup metadata: checksum, size, storage type, status, and audit trail.

Explicitly excluded in P3:

- Raw secret export.
- Arbitrary filesystem backup.
- Docker volume backup.
- Remote object storage.
- Full host bare-metal recovery.

## Pre-Restore Inputs

Before starting a restore, collect:

- A verified Postgres dump.
- The backup checksum and size from SnowPanel backup metadata.
- The SnowPanel deployment commit or release version.
- Fresh production secrets for JWT, encryption, database password, and agent auth token.
- The expected core-agent allowlists, service whitelist, cron allowlist, and allowed roots.

Do not reuse leaked or unknown secrets from a backup artifact.

## Fresh Machine Restore

1. Provision a fresh host with Docker, Docker Compose, PowerShell 7, Go, Node.js, Rust, and the SnowPanel repository.
2. Check out the release or commit recorded with the backup metadata.
3. Create fresh `.env` values from `.env.example`.
4. Restore observability and core-agent config snapshots only after reviewing host-specific paths and endpoints.
5. Start Postgres without the backend first.

## Restore Postgres

Example command shape:

```bash
psql "$DATABASE_URL" < snowpanel-postgres.dump.sql
```

After restore:

- Confirm migrations are at the expected level.
- Confirm the `users`, `roles`, `permissions`, `role_permissions`, `tasks`, `task_logs`, `audit_logs`, and `backups` tables exist.
- Confirm no encrypted setting is read without `SNOWPANEL_ENCRYPTION_KEY`.

## Rotate Secrets

Rotate these before exposing the restored service:

- `JWT_SECRET`
- `SNOWPANEL_ENCRYPTION_KEY`
- `BACKEND_AGENT_SHARED_TOKEN`
- `CORE_AGENT_SHARED_TOKEN`
- Database password if the restore reused old database credentials.

If encrypted settings were restored, keep the old encryption key only long enough to decrypt and re-encrypt settings with the new key.

## Start Services

Start backend/frontend:

```bash
make up
```

For host-agent mode:

```bash
make up-host-agent
```

## Verify Health

Run:

```bash
curl -f http://127.0.0.1:8080/health
curl -f http://127.0.0.1:8080/ready
```

Expected result:

- `/health` returns healthy.
- `/ready` returns ready.
- Backend logs do not report missing encryption key, unsafe production agent auth, or database migration errors.

## Login Verification

1. Sign in with an administrator account.
2. Change any bootstrap or emergency password immediately.
3. Confirm the session includes expected permissions.
4. Confirm protected pages load without 401/403 errors.

## Audit Verification

After login, verify:

- New login and configuration actions appear in audit logs.
- `request_id` is present on newly created audit records.
- Audit export still works for CSV and JSONL.
- Audit retention cleanup remains disabled unless explicitly triggered by an operator.

## Task and Backup Verification

Run a non-destructive task check:

- List tasks.
- Open one historical task detail.
- Confirm task logs are visible.

For backup foundation validation:

- Verify backup metadata rows have checksum, size, status, storage type, and resource fields.
- Confirm local artifacts are stored under `BACKUP_LOCAL_DIR` and not under a public web root.
- Recompute checksum for a restored backup artifact and compare it to metadata.
- Create a fresh backup after restore and verify it before allowing destructive operations.

## Rollback

If restore validation fails:

1. Stop backend/frontend.
2. Preserve backend logs and Postgres logs.
3. Record the failing command and request id.
4. Recreate the database from a known-good dump.
5. Do not continue with destructive operations until `/health`, `/ready`, login, audit, and backup verification pass.
