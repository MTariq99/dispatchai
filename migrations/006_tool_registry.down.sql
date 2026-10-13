-- migrations/006_tool_registry.down.sql
DROP TABLE IF EXISTS tool_registration_idempotency;

DROP TABLE IF EXISTS tool_definition_versions;

DROP TABLE IF EXISTS tool_definitions;

DROP TABLE IF EXISTS host_integrations;
