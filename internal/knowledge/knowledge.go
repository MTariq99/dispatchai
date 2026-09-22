package knowledge

// This file defines the optional knowledge/RAG abstraction.
//
// Knowledge retrieval can provide information such as:
//
//   - company policies
//   - SOPs
//   - documentation
//   - manuals
//   - internal procedures
//
// Knowledge retrieval is NOT the source of live business state.
//
// If the Assistant needs to know the current status of an order,
// vehicle, technician, customer, etc., it should request that data
// through a Host Project tool.
