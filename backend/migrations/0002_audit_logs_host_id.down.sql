DROP INDEX IF EXISTS idx_audit_logs_host_id;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS host_id;
