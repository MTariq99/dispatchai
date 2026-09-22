package llm

// This file defines the internal representation of an LLM request.
//
// A request may contain:
//
//   - system instructions
//   - conversation messages
//   - available tool definitions
//   - user context
//   - optional knowledge context
//   - model configuration
//
// The request represents what the Assistant wants the LLM to reason
// about.
//
// It does not contain direct database access or business services.
