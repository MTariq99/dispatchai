-- migrations/009_tool_registry_integrity.up.sql
-- Required so a foreign key can verify both tool ID and owner integration.
ALTER TABLE tool_definitions
ADD CONSTRAINT uq_tool_definitions_integration_id_id UNIQUE (integration_id, id);

-- The registration request hash must be a SHA-256 digest.
ALTER TABLE tool_registration_idempotency
ADD CONSTRAINT tool_registration_idempotency_hash_length CHECK (octet_length(request_hash) = 32);

-- An idempotency record must point to a tool owned by the same integration.
ALTER TABLE tool_registration_idempotency
ADD CONSTRAINT fk_tool_registration_same_integration FOREIGN KEY (integration_id, tool_id) REFERENCES tool_definitions (integration_id, id) ON DELETE RESTRICT;
