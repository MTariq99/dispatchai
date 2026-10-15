CREATE TABLE integration_request_nonces (
    integration_id UUID NOT NULL REFERENCES host_integrations (id) ON DELETE CASCADE,
    nonce VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (integration_id, nonce)
);

CREATE INDEX idx_integration_request_nonces_expires ON integration_request_nonces (expires_at);