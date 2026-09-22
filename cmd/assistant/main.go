package main

// This file is the entry point for the reusable AI Assistant service.
//
// Its job is ONLY application startup and dependency wiring.
//
// It loads configuration, initializes infrastructure and adapters,
// creates the Assistant and its dependencies, registers the API routes,
// and starts the HTTP server.
//
// It does NOT contain:
//   - LLM reasoning logic
//   - tool execution logic
//   - business logic
//   - notification business rules
//   - database queries
//
// The runtime dependency chain is:
//
//   Config
//      ↓
//   LLM Client
//   Project Client
//   Memory
//   Policy Engine
//   Audit Store
//   Observability
//      ↓
//   Assistant
//      ↓
//   HTTP API
//
// The actual business operations remain inside the Host Project.
