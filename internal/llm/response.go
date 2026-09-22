package llm

// This file defines the normalized response returned by an LLM.
//
// An LLM response can contain:
//
//   - normal text
//   - one or more structured tool calls
//   - token usage
//   - provider metadata
//   - finish information
//
// The Assistant uses this normalized representation without needing
// to know which LLM provider generated it.
//
// A response containing a tool call means:
//
//   "The model wants the Host Project to perform this operation."
//
// It does NOT mean the model has executed the operation.
