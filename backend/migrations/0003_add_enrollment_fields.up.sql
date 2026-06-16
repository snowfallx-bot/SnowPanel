-- Add enrollment fields to hosts table for mTLS enrollment management
ALTER TABLE hosts
    ADD COLUMN IF NOT EXISTS enrollment_id VARCHAR(128) UNIQUE,
    ADD COLUMN IF NOT EXISTS agent_cert_hash VARCHAR(64),
    ADD COLUMN IF NOT EXISTS revoked BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS revoked_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_hosts_enrollment_id ON hosts (enrollment_id);
CREATE INDEX IF NOT EXISTS idx_hosts_revoked ON hosts (revoked);
