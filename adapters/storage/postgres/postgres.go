package postgres

// This package contains Postgres implementations of Assistant-owned
// persistence interfaces.
//
// Possible data includes:
//
//   - conversations
//   - audit events
//   - approvals
//   - idempotency records
//
// It must NOT contain Host Project domain tables such as:
//
//   - orders
//   - vehicles
//   - drivers
//   - patients
//   - technicians
//
// Those belong to the Host Project.
