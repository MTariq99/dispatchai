package policy

// This file implements policy evaluation.
//
// It evaluates requested operations against deterministic rules and
// produces a policy decision.
//
// Possible decisions:
//
//   ALLOW
//   DENY
//   REQUIRE_APPROVAL
//
// Example:
//
//   LLM requests sending a notification to 50,000 recipients.
//
//   Policy Engine
//        ↓
//   Recipient limit exceeded
//        ↓
//   REQUIRE_APPROVAL
//
// The LLM cannot override this decision.
