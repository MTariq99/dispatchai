package openai

// This package contains the concrete OpenAI integration.
//
// It implements the internal LLM interface using OpenAI's API.
//
// Provider-specific concerns belong here:
//
//   - authentication
//   - request serialization
//   - tool schema conversion
//   - response parsing
//   - provider errors
//   - usage information
//
// The Assistant itself must not depend on OpenAI-specific types.
