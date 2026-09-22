package approval

// This file defines the lifecycle states of an approval request.
//
// Typical states are:
//
//   PENDING
//   APPROVED
//   REJECTED
//   EXPIRED
//   CANCELLED
//
// These states allow the rest of the system to determine whether an
// action is allowed to continue.
