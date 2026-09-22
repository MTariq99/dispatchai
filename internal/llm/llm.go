package llm

// This file defines the generic LLM interface used by the Assistant.
//
// The Assistant depends on this abstraction rather than directly
// depending on OpenAI, Gemini, Anthropic, or another provider.
//
// The interface represents operations such as:
//
//   Generate(request)
//
// Provider-specific implementations translate between this internal
// representation and the provider's API.
//
// This separation allows the LLM provider to be changed without
// rewriting the Assistant orchestration layer.
