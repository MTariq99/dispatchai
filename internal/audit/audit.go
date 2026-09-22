package audit

// This file defines the audit logging contract.
//
// Important Assistant actions should be auditable, including:
//
//   - requests
//   - LLM tool calls
//   - policy decisions
//   - approval requests
//   - tool executions
//   - notification operations
//   - failures
//
// Audit records provide an operational history of what the system
// attempted and what actually happened.
