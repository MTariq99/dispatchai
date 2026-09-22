package llm

// This file contains the OpenAI-specific implementation of the generic
// LLM interface.
//
// Its responsibility is translation:
//
//   Internal LLM Request
//        ↓
//   OpenAI API Request
//        ↓
//   OpenAI Response
//        ↓
//   Internal LLM Response
//
// Provider-specific JSON formats, authentication, error handling,
// tool-call parsing, and token usage conversion belong here.
//
// The Assistant orchestration layer must remain provider-independent.
