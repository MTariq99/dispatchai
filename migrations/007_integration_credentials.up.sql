CREATE TABLE integration_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_id UUID NOT NULL REFERENCES host_integrations (id) ON DELETE RESTRICT,
    token_hash BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    CONSTRAINT integration_credentials_hash_length CHECK (octet_length(token_hash) = 32)
);

CREATE INDEX idx_integration_credentials_active ON integration_credentials (integration_id)
WHERE
    revoked_at IS NULL;
