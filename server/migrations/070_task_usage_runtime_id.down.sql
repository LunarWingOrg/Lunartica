ALTER TABLE task_usage DROP CONSTRAINT IF EXISTS task_usage_runtime_id_fkey;
ALTER TABLE task_usage DROP CONSTRAINT IF EXISTS task_usage_runtime_id_not_null;
ALTER TABLE task_usage DROP COLUMN IF EXISTS runtime_id;
