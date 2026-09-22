package tools

// This file is responsible for forwarding LLM-generated tool calls
// to the Host Project.
//
// It does NOT execute tools directly.
//
// The execution path is:
//
//   LLM
//      ↓
//   Tool Call
//      ↓
//   Tool Executor
//      ↓
//   Project Client
//      ↓
//   Host Project
//      ↓
//   Business Service / Database
//      ↓
//   Tool Result
//
// The executor validates the call, sends it across the project boundary,
// handles communication errors, and converts the response into the
// internal ToolResult representation.
//
// This boundary is critical to keeping DispatchAI domain-agnostic.
