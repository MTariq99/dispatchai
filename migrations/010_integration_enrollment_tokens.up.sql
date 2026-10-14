CREATE TABLE integration_enrollment_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    integration_id UUID NOT NULL
        REFERENCES host_integrations (id)
        ON DELETE RESTRICT,

    token_hash BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,

    CONSTRAINT integration_enrollment_token_hash_length
        CHECK (octet_length(token_hash) = 32),

    CONSTRAINT integration_enrollment_token_expiry
        CHECK (expires_at > created_at),

    CONSTRAINT integration_enrollment_token_consumption
        CHECK (consumed_at IS NULL OR consumed_at >= created_at)
);

CREATE INDEX idx_integration_enrollment_tokens_pending
    ON integration_enrollment_tokens (integration_id, expires_at)
    WHERE consumed_at IS NULL;
