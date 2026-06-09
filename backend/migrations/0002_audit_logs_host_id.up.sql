ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS host_id BIGINT REFERENCES hosts(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_host_id ON audit_logs (host_id);
