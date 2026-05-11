ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS request_id VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS trace_id VARCHAR(64) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs (request_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_trace_id ON audit_logs (trace_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_target ON audit_logs (target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_result_code ON audit_logs (result_code);
