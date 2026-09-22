package project

// This file defines the request sent from DispatchAI to the Host
// Project for tool execution.
//
// The request can contain:
//
//   - tool name
//   - tool arguments
//   - request ID
//   - conversation ID
//   - authenticated user identity
//   - tenant context
//   - authorization context
//   - tracing metadata
//
// These fields allow the Host Project to apply its own security and
// business policies before executing the requested operation.
