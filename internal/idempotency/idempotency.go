package idempotency

// This file defines the idempotency contract used to prevent duplicate
// execution.
//
// This is especially important for operations such as notifications.
//
// Example:
//
//   Request:
//       send_notification(id=123)
//
//   Network timeout occurs.
//
//   Assistant retries.
//
// Without idempotency:
//
//   Notification 1 -> sent
//   Notification 2 -> sent again
//
// With idempotency:
//
//   Existing execution detected
//        ↓
//   Duplicate execution prevented
//
// The actual business operation remains Host Project-owned.
