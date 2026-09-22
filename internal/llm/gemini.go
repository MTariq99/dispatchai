package llm

// This file contains the Gemini-specific implementation of the generic
// LLM interface.
//
// It translates the Assistant's internal request format into Gemini's
// API format and converts Gemini responses back into the application's
// normalized LLM response.
//
// Provider-specific behavior belongs here so the rest of the system
// does not depend on Gemini-specific types.
