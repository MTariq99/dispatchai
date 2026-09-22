package redis

// This package contains Redis implementations for short-lived or
// high-speed Assistant state.
//
// Possible uses include:
//
//   - caching
//   - rate limiting
//   - locks
//   - temporary state
//   - short-lived idempotency data
//
// Durable business state remains outside this package.
