DROP INDEX IF EXISTS idx_tasks_idempotency_key;
DROP INDEX IF EXISTS idx_tasks_claim;

ALTER TABLE tasks DROP COLUMN IF EXISTS next_run_at;
ALTER TABLE tasks DROP COLUMN IF EXISTS idempotency_key;
ALTER TABLE tasks DROP COLUMN IF EXISTS max_attempts;
ALTER TABLE tasks DROP COLUMN IF EXISTS attempt;
ALTER TABLE tasks DROP COLUMN IF EXISTS locked_until;
ALTER TABLE tasks DROP COLUMN IF EXISTS locked_by;
