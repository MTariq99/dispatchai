package pgvector

// This package implements the generic Knowledge Provider using
// PostgreSQL with pgvector.
//
// The flow is:
//
//   Assistant
//       ↓
//   Knowledge Interface
//       ↓
//   pgvector Adapter
//       ↓
//   PostgreSQL
//       ↓
//   Retrieved Knowledge
//       ↓
//   LLM Context
//
// This is for knowledge retrieval, not authoritative live business
// state.
