package assistant

// This file implements the repeated LLM -> Tool -> Result -> LLM loop.
//
// The LLM may require several tool calls before it can produce a final
// answer.
//
// Example:
//
//   LLM
//    ↓
//   find_delayed_orders
//    ↓
//   Tool Result
//    ↓
//   LLM
//    ↓
//   get_customer_preferences
//    ↓
//   Tool Result
//    ↓
//   LLM
//    ↓
//   send_notification
//    ↓
//   Tool Result
//    ↓
//   LLM
//    ↓
//   Final Answer
//
// This file also protects the system from uncontrolled execution by
// enforcing limits such as:
//
//   - maximum iterations
//   - maximum tool calls
//   - execution timeout
//   - invalid tool calls
//   - repeated tool calls
//
// It never directly executes business operations.
