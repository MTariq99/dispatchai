DROP INDEX IF EXISTS idx_conversations_integration;

ALTER TABLE conversations
DROP COLUMN IF EXISTS integration_id;
