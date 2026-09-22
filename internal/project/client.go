package project

// This file provides the client used by DispatchAI to communicate with
// the Host Project.
//
// Depending on the integration, this may use HTTP or gRPC.
//
// Its responsibilities include:
//
//   - sending tool execution requests
//   - retrieving available capabilities
//   - health checking the Host Project
//   - handling transport-level failures
//
// It does NOT contain business logic.
//
// It is simply the communication bridge:
//
//   DispatchAI <-> Host Project
