package llm

// This file provides a fake LLM implementation for tests.
//
// Tests should not need to make real LLM API calls.
//
// The mock can return predetermined responses such as:
//
//   1. Tool call
//   2. Another tool call
//   3. Final response
//
// This allows deterministic testing of the Assistant orchestration,
// tool loop, error handling, and policy behavior.
