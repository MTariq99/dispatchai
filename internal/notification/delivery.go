package notification

// This file represents notification delivery state.
//
// A notification may move through states such as:
//
//   QUEUED
//      ↓
//   SENDING
//      ↓
//   ACCEPTED
//      ↓
//   DELIVERED
//
// or:
//
//   SENDING
//      ↓
//   FAILED
//
// This distinction is important because "accepted by provider" is not
// necessarily the same as "delivered to the recipient."
