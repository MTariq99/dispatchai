package idempotency

// This file defines persistence for idempotency records.
//
// Records allow the system to remember that a particular logical
// operation has already been reserved, completed, or failed.
//
// Storage may be implemented using Postgres or Redis depending on the
// consistency and lifetime requirements.
