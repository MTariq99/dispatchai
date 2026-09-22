package config

// This file registers the HTTP routes exposed by DispatchAI.
//
// Example routes:
//
//   POST /v1/assistant/chat
//   POST /v1/assistant/dispatch
//   POST /v1/assistant/events
//   GET  /v1/assistant/conversations/:id
//
// Route registration belongs here so the HTTP entry surface remains
// easy to discover and maintain.
