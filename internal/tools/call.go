package tools

// This file represents an invocation request for a Host Project tool.
//
// The call normally contains:
//
//   - tool name
//   - structured arguments
//   - request metadata
//   - correlation information
//
// The LLM creates the logical request, but the application validates
// it before forwarding it to the Host Project.
