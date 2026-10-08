CREATE TABLE conversations (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    title TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ
);

CREATE INDEX idx_conversations_tenant_user ON conversations (tenant_id, user_id, updated_at DESC);

CREATE TABLE conversation_messages (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    sequence BIGINT NOT NULL,
    role VARCHAR(32) NOT NULL,
    content TEXT,
    tool_calls JSONB,
    tool_call_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (conversation_id, sequence)
);

CREATE INDEX idx_conversation_messages_conversation ON conversation_messages (conversation_id, sequence);

CREATE TABLE conversation_runs (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    request_id UUID NOT NULL,
    status VARCHAR(32) NOT NULL,
    model_provider VARCHAR(64),
    model_name VARCHAR(128),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    error_code TEXT,
    error_message TEXT
);

CREATE INDEX idx_conversation_runs_conversation ON conversation_runs (conversation_id, started_at DESC);

CREATE UNIQUE INDEX uq_conversation_runs_request ON conversation_runs (request_id);

CREATE TABLE tool_executions (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES conversation_runs (id) ON DELETE CASCADE,
    call_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    arguments JSONB NOT NULL,
    status VARCHAR(32) NOT NULL,
    result JSONB,
    error_code TEXT,
    error_message TEXT,
    retryable BOOLEAN NOT NULL DEFAULT FALSE,
    attempt INT NOT NULL DEFAULT 1,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    UNIQUE (run_id, call_id)
);

CREATE INDEX idx_tool_executions_run ON tool_executions (run_id);

CREATE INDEX idx_tool_executions_tool ON tool_executions (tool_name);

CREATE TABLE idempotency_keys (
    key TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    conversation_id UUID NOT NULL,
    request_id UUID NOT NULL,
    run_id UUID,
    operation VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, key),
    UNIQUE (request_id),
    CONSTRAINT fk_idempotency_run FOREIGN KEY (run_id) REFERENCES conversation_runs (id) ON DELETE SET NULL
);

CREATE INDEX idx_idempotency_conversation ON idempotency_keys (tenant_id, conversation_id);

CREATE INDEX idx_idempotency_run ON idempotency_keys (run_id);

CREATE INDEX idx_idempotency_expires ON idempotency_keys (expires_at);

CREATE INDEX idx_idempotency_tenant ON idempotency_keys (conversation_id);

CREATE TABLE audit_events (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    request_id UUID,
    conversation_id UUID,
    run_id UUID,
    actor_type VARCHAR(32) NOT NULL,
    actor_id UUID,
    event_type VARCHAR(128) NOT NULL,
    resource_type VARCHAR(128),
    resource_id TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_tenant_time ON audit_events (tenant_id, created_at DESC);

CREATE INDEX idx_audit_events_request ON audit_events (request_id);

CREATE INDEX idx_audit_events_conversation ON audit_events (conversation_id);

CREATE TABLE approvals (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    conversation_id UUID,
    run_id UUID,
    tool_execution_id UUID,
    requested_by UUID NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    reason TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ,
    decided_by UUID
);

CREATE INDEX idx_approvals_pending ON approvals (tenant_id, status);

CREATE INDEX idx_approvals_run ON approvals (run_id);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    aggregate_type VARCHAR(128) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    attempts INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_outbox_pending ON outbox_events (status, created_at);