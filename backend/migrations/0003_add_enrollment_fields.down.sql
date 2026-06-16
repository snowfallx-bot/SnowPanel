-- Remove enrollment fields from hosts table
ALTER TABLE hosts
    DROP COLUMN IF EXISTS enrollment_id,
    DROP COLUMN IF EXISTS agent_cert_hash,
    DROP COLUMN IF EXISTS revoked,
    DROP COLUMN IF EXISTS revoked_at,
    DROP COLUMN IF EXISTS revoked_reason;
