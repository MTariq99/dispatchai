package assistant

// This file contains the main orchestration logic of the AI Assistant.
//
// It coordinates the complete reasoning cycle between the LLM,
// conversation state, tool executor, and Host Project.
//
// The normal flow is:
//
//   User Request
//        ↓
//   Build Context
//        ↓
//   Send Request + Tools to LLM
//        ↓
//   LLM returns Tool Call
//        ↓
//   Forward Tool Call to Host Project
//        ↓
//   Receive Tool Result
//        ↓
//   Give Result back to LLM
//        ↓
//   More Tool Calls OR Final Answer
//
// This is the central control plane of the Assistant.
//
// It does NOT execute business operations itself.
//
// The Host Project remains responsible for:
//   - authorization
//   - business validation
//   - database access
//   - business services
//   - external business APIs
//   - actual notification delivery
