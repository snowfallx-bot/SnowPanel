# Migrations

SQL migration files for PostgreSQL.

## Files

- `0001_init_schema.up.sql`: creates baseline schema
- `0001_init_schema.down.sql`: drops baseline schema

## Apply (example with psql)

`psql "host=127.0.0.1 port=5432 dbname=snowpanel user=snowpanel password=snowpanel sslmode=disable" -f backend/migrations/0001_init_schema.up.sql`

## Rollback

`psql "host=127.0.0.1 port=5432 dbname=snowpanel user=snowpanel password=snowpanel sslmode=disable" -f backend/migrations/0001_init_schema.down.sql`
