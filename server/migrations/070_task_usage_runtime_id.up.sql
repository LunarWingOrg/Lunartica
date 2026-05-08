ALTER TABLE task_usage ADD COLUMN IF NOT EXISTS runtime_id UUID;

UPDATE task_usage tu
SET runtime_id = atq.runtime_id
FROM agent_task_queue atq
WHERE atq.id = tu.task_id
  AND tu.runtime_id IS NULL;

ALTER TABLE task_usage
    ADD CONSTRAINT task_usage_runtime_id_not_null
    CHECK (runtime_id IS NOT NULL)
    NOT VALID;

ALTER TABLE task_usage VALIDATE CONSTRAINT task_usage_runtime_id_not_null;

ALTER TABLE task_usage
    ADD CONSTRAINT task_usage_runtime_id_fkey
    FOREIGN KEY (runtime_id) REFERENCES agent_runtime(id) ON DELETE CASCADE
    NOT VALID;

ALTER TABLE task_usage VALIDATE CONSTRAINT task_usage_runtime_id_fkey;
