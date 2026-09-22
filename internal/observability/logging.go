package observability

// This file provides structured application logging.
//
// Logs should contain correlation information such as:
//
//   - request_id
//   - conversation_id
//   - tenant_id
//   - tool_name
//   - trace_id
//
// Logs should describe what the system did without exposing sensitive
// user information, credentials, tokens, or unrestricted LLM payloads.
