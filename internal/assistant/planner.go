package assistant

// This file contains planning logic for requests that require multiple
// logical steps.
//
// Example:
//
//   "Find delayed customers and notify them."
//
// The Assistant may need to:
//
//   1. Find matching entities.
//   2. Retrieve recipient information.
//   3. Retrieve communication preferences.
//   4. Request notification execution.
//
// The planner determines what needs to happen and in what order.
//
// It does NOT execute those operations.
//
// Planning answers:
//
//   "What should happen next?"
//
// The Host Project answers:
//
//   "How is that operation actually performed?"
