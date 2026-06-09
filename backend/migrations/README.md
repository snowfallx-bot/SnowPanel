# Migrations

SQL migration files for PostgreSQL.

## Files

- `0001_init_schema.up.sql`: creates baseline schema
- `0001_init_schema.down.sql`: drops baseline schema
- `0002_audit_logs_host_id.up.sql`: backfills the new `audit_logs.host_id` column for existing databases
- `0002_audit_logs_host_id.down.sql`: removes the `audit_logs.host_id` upgrade

## Apply (example with psql)

`psql "host=127.0.0.1 port=5432 dbname=snowpanel user=snowpanel password=snowpanel sslmode=disable" -f backend/migrations/0001_init_schema.up.sql`

For an existing database created before `0002`, apply the upgrade after `0001`:

`psql "host=127.0.0.1 port=5432 dbname=snowpanel user=snowpanel password=snowpanel sslmode=disable" -f backend/migrations/0002_audit_logs_host_id.up.sql`

## Rollback

`psql "host=127.0.0.1 port=5432 dbname=snowpanel user=snowpanel password=snowpanel sslmode=disable" -f backend/migrations/0001_init_schema.down.sql`
