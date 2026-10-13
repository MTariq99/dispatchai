-- migrations/006_tool_registry.up.sql
CREATE TABLE host_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_key VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('pending', 'active', 'suspended')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tool_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_id UUID NOT NULL REFERENCES host_integrations (id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    input_schema JSONB NOT NULL DEFAULT '{"type":"object","properties":{}}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT tool_definitions_version_positive CHECK (version > 0),
    CONSTRAINT tool_definitions_name_nonempty CHECK (length(trim(name)) > 0),
    CONSTRAINT tool_definitions_schema_object CHECK (
        jsonb_typeof(input_schema) = 'object'
        AND input_schema ->> 'type' = 'object'
    )
);

-- Retired names may be reused, but active names must be unique
-- within a host integration.
CREATE UNIQUE INDEX uq_tool_definitions_active_name ON tool_definitions (integration_id, name)
WHERE
    deleted_at IS NULL;

CREATE INDEX idx_tool_definitions_available ON tool_definitions (integration_id, name)
WHERE
    enabled = TRUE
    AND deleted_at IS NULL;

-- Historical snapshots are never updated in place.
CREATE TABLE tool_definition_versions (
    tool_id UUID NOT NULL REFERENCES tool_definitions (id) ON DELETE RESTRICT,
    version INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    input_schema JSONB NOT NULL,
    enabled BOOLEAN NOT NULL,
    change_type VARCHAR(20) NOT NULL CHECK (
        change_type IN (
            'created',
            'updated',
            'enabled',
            'disabled',
            'deleted'
        )
    ),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tool_id, version)
);

-- Idempotency for tool creation requests.
CREATE TABLE tool_registration_idempotency (
    integration_id UUID NOT NULL REFERENCES host_integrations (id) ON DELETE RESTRICT,
    idempotency_key VARCHAR(255) NOT NULL,
    request_hash BYTEA NOT NULL,
    tool_id UUID NOT NULL REFERENCES tool_definitions (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (integration_id, idempotency_key)
);
