package tools

// This file maintains the set of tool definitions available to the
// Assistant for a request or Host Project.
//
// This registry is NOT a registry of business implementations owned by
// DispatchAI.
//
// The Host Project provides the available capabilities.
//
// DispatchAI stores those definitions so they can be:
//
//   - validated
//   - looked up
//   - supplied to the LLM
//   - checked when the LLM returns a tool call
//
// The actual execution still happens inside the Host Project.
