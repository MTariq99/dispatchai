-- migrations/008_integration_signing_keys.up.sql
-- X-Client-Id maps to host_integrations.integration_key.
-- X-Key-Id maps to integration_keys.key_id.
-- Only public keys are stored by DispatchAI.
CREATE TABLE IF NOT EXISTS integration_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_id UUID NOT NULL REFERENCES host_integrations (id) ON DELETE RESTRICT,
    key_id VARCHAR(128) NOT NULL,
    algorithm VARCHAR(32) NOT NULL DEFAULT 'ECDSA-P256-SHA256' CHECK (algorithm = 'ECDSA-P256-SHA256'),
    -- Uncompressed SEC1 P-256 point: 0x04 || X || Y.
    -- Go must additionally validate that this is a valid P-256 point.
    public_key BYTEA NOT NULL CHECK (
        octet_length(public_key) = 65
        AND get_byte(public_key, 0) = 4
    ),
    status VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT uq_integration_keys_integration_key UNIQUE (integration_id, key_id),
    CONSTRAINT integration_keys_revocation_consistent CHECK (
        (
            status = 'active'
            AND revoked_at IS NULL
        )
        OR (
            status = 'revoked'
            AND revoked_at IS NOT NULL
        )
    ),
    CONSTRAINT integration_keys_expiry_after_creation CHECK (
        expires_at IS NULL
        OR expires_at > created_at
    )
);

CREATE INDEX idx_integration_keys_active_lookup ON integration_keys (integration_id, key_id)
WHERE
    status = 'active'
    AND revoked_at IS NULL;

-- The catalog defines valid permissions.
-- Grants determine which permissions an integration actually has.
CREATE TABLE IF NOT EXISTS authorization_permissions (
    permission VARCHAR(100) PRIMARY KEY,
    description TEXT NOT NULL
);

INSERT INTO
    authorization_permissions (permission, description)
VALUES
    (
        'assistant:chat',
        'Submit assistant chat requests'
    ),
    ('tools:register', 'Register tool definitions'),
    ('tools:read', 'Read owned tool definitions'),
    ('tools:update', 'Update owned tool definitions'),
    (
        'tools:enable',
        'Enable or disable owned tool definitions'
    ),
    (
        'tools:delete',
        'Soft-delete owned tool definitions'
    );

CREATE TABLE integration_permission_grants (
    integration_id UUID NOT NULL REFERENCES host_integrations (id) ON DELETE RESTRICT,
    permission VARCHAR(100) NOT NULL REFERENCES authorization_permissions (permission) ON DELETE RESTRICT,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by VARCHAR(200),
    PRIMARY KEY (integration_id, permission)
);

CREATE INDEX idx_integration_permission_grants_permission ON integration_permission_grants (permission, integration_id);
