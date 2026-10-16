BEGIN;

-- 1. Idempotency keys
ALTER TABLE idempotency_keys
ADD COLUMN integration_id UUID REFERENCES host_integrations (id) ON DELETE RESTRICT;

UPDATE idempotency_keys AS ik
SET
    integration_id = c.integration_id
FROM
    conversations AS c
WHERE
    c.id = ik.conversation_id;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM idempotency_keys
        WHERE integration_id IS NULL
    ) THEN
        RAISE EXCEPTION
            'Cannot migrate idempotency_keys: integration ownership is missing';
    END IF;
END
$$;

ALTER TABLE idempotency_keys
DROP CONSTRAINT idempotency_keys_pkey,
DROP COLUMN tenant_id,
ALTER COLUMN integration_id
SET NOT NULL,
ADD CONSTRAINT idempotency_keys_pkey PRIMARY KEY (integration_id, key);

DROP INDEX IF EXISTS idx_idempotency_conversation;

DROP INDEX IF EXISTS idx_idempotency_tenant;

CREATE INDEX idx_idempotency_integration_conversation ON idempotency_keys (integration_id, conversation_id);

-- 2. Audit events
ALTER TABLE audit_events
ADD COLUMN integration_id UUID REFERENCES host_integrations (id) ON DELETE RESTRICT;

UPDATE audit_events AS ae
SET
    integration_id = c.integration_id
FROM
    conversations AS c
WHERE
    c.id = ae.conversation_id;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM audit_events
        WHERE integration_id IS NULL
    ) THEN
        RAISE EXCEPTION
            'Cannot migrate audit_events: integration ownership is missing';
    END IF;
END
$$;

DROP INDEX IF EXISTS idx_audit_events_tenant_time;

ALTER TABLE audit_events
DROP COLUMN tenant_id,
ALTER COLUMN integration_id
SET NOT NULL;

CREATE INDEX idx_audit_events_integration_time ON audit_events (integration_id, created_at DESC);

-- 3. Approvals
ALTER TABLE approvals
ADD COLUMN integration_id UUID REFERENCES host_integrations (id) ON DELETE RESTRICT;

UPDATE approvals AS a
SET
    integration_id = c.integration_id
FROM
    conversation_runs AS r
    JOIN conversations AS c ON c.id = r.conversation_id
WHERE
    a.run_id = r.id
    AND a.integration_id IS NULL;

UPDATE approvals AS a
SET
    integration_id = c.integration_id
FROM
    conversations AS c
WHERE
    a.conversation_id = c.id
    AND a.integration_id IS NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM approvals
        WHERE integration_id IS NULL
    ) THEN
        RAISE EXCEPTION
            'Cannot migrate approvals: integration ownership is missing';
    END IF;
END
$$;

DROP INDEX IF EXISTS idx_approvals_pending;

ALTER TABLE approvals
DROP COLUMN tenant_id,
ALTER COLUMN integration_id
SET NOT NULL;

CREATE INDEX idx_approvals_integration_status ON approvals (integration_id, status);

-- 4. Outbox events
ALTER TABLE outbox_events
ADD COLUMN integration_id UUID REFERENCES host_integrations (id) ON DELETE RESTRICT;

-- There is no ownership backfill in the existing schema.
-- Refuse to discard legacy ownership if any rows remain.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM outbox_events
        WHERE integration_id IS NULL
    ) THEN
        RAISE EXCEPTION
            'Cannot migrate outbox_events: integration ownership is missing';
    END IF;
END
$$;

DROP INDEX IF EXISTS idx_outbox_pending;

ALTER TABLE outbox_events
DROP COLUMN tenant_id,
ALTER COLUMN integration_id
SET NOT NULL;

CREATE INDEX idx_outbox_integration_status_created ON outbox_events (integration_id, status, created_at);

COMMIT;
