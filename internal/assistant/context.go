package assistant

// This file builds the context supplied to the LLM.
//
// Context may include:
//
//   - authenticated user identity
//   - tenant information
//   - roles or permission context
//   - conversation history
//   - request metadata
//   - available Host Project capabilities
//   - optional knowledge retrieved through RAG
//
// The identity and authorization context comes from the Host Project.
// The LLM must never be allowed to invent or modify those values.
//
// This file prepares information for reasoning; it does not make
// authorization decisions.
