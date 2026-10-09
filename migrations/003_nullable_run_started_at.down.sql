UPDATE conversation_runs
SET started_at = COALESCE(created_at, NOW())
WHERE started_at IS NULL;

ALTER TABLE conversation_runs
ALTER COLUMN started_at SET NOT NULL;
