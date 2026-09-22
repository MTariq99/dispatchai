package policy

// This file contains deterministic safety and business-independent
// protection rules used by the Assistant.
//
// Examples include:
//
//   - maximum recipient count
//   - maximum tool calls
//   - allowed notification channels
//   - rate limits
//   - tenant restrictions
//   - approval thresholds
//   - execution limits
//
// These rules exist because an LLM must never be treated as the final
// authority for sensitive or expensive operations.
