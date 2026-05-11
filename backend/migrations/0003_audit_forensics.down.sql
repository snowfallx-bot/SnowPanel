DROP INDEX IF EXISTS idx_audit_logs_result_code;
DROP INDEX IF EXISTS idx_audit_logs_target;
DROP INDEX IF EXISTS idx_audit_logs_trace_id;
DROP INDEX IF EXISTS idx_audit_logs_request_id;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS trace_id,
    DROP COLUMN IF EXISTS request_id;
