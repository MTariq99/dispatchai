package audit

// This file defines persistence operations for audit events.
//
// It provides methods for saving and querying audit information.
//
// Queries may include:
//
//   - events for a request
//   - events for a conversation
//   - events for a tenant
//
// The storage implementation belongs behind this abstraction.
