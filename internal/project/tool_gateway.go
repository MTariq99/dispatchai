package project

// This file defines the protocol boundary used when DispatchAI asks
// the Host Project to execute a capability.
//
// The gateway is deliberately generic.
//
// DispatchAI says:
//
//   "Execute tool X with these arguments."
//
// The Host Project decides:
//
//   - whether the caller is authorized
//   - whether the arguments are valid
//   - whether business rules permit the operation
//   - how to access its database/services
//   - how to perform the operation
//
// This file therefore represents the most important integration
// boundary in the system.
