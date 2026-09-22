package knowledge

// This file defines the generic knowledge provider interface.
//
// DispatchAI does not depend on a specific vector database.
//
// Possible implementations include:
//
//   - pgvector
//   - external vector database
//   - knowledge API
//   - another retrieval service
//
// The Assistant only knows how to ask for relevant knowledge.
