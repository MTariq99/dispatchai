package observability

// This file provides distributed tracing support.
//
// A single request should be traceable across:
//
//   Host Project
//        ↓
//   AI Assistant
//        ↓
//   LLM
//        ↓
//   Tool Executor
//        ↓
//   Host Project
//        ↓
//   Business Service
//
// Tracing makes latency and failure boundaries visible.
