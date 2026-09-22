package notification

// This file coordinates notification dispatch behavior.
//
// It can represent operations such as:
//
//   - dispatching one notification
//   - dispatching multiple notifications
//   - selecting a requested channel
//   - handling delivery results
//   - handling failures
//
// IMPORTANT:
//
// Under the core DispatchAI architecture, actual business notification
// execution belongs to the Host Project.
//
// Therefore this package must not bypass the project boundary and send
// messages directly when the Host Project owns notification delivery.
