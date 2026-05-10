ALTER TABLE tasks ADD COLUMN IF NOT EXISTS locked_by TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS attempt INT NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS max_attempts INT NOT NULL DEFAULT 1;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_tasks_claim ON tasks (status, next_run_at, locked_until);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_idempotency_key ON tasks (idempotency_key) WHERE idempotency_key IS NOT NULL;
