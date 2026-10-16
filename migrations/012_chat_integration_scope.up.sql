BEGIN;

ALTER TABLE conversations
ADD COLUMN IF NOT EXISTS integration_id UUID REFERENCES host_integrations (id) ON DELETE RESTRICT;

DROP INDEX IF EXISTS idx_conversations_tenant_user;

ALTER TABLE conversations
DROP COLUMN tenant_id,
DROP COLUMN user_id;

ALTER TABLE conversations
ALTER COLUMN integration_id
SET NOT NULL;

CREATE INDEX idx_conversations_integration ON conversations (integration_id);

COMMIT;
