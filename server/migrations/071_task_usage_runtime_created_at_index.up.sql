CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_usage_runtime_created_at
    ON task_usage (runtime_id, created_at DESC)
    INCLUDE (
        task_id,
        provider,
        model,
        input_tokens,
        output_tokens,
        cache_read_tokens,
        cache_write_tokens
    );
