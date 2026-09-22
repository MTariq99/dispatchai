package api

// This file contains the HTTP handlers for the Assistant API.
//
// Handlers are intentionally thin.
//
// Their responsibility is:
//
//   HTTP Request
//      ↓
//   Validate/Decode
//      ↓
//   Call Assistant
//      ↓
//   Encode Response
//
// They should not contain LLM reasoning, tool execution, business
// logic, or database queries.
