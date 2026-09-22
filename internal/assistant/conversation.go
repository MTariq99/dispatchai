package assistant

// This file manages the conversation state used during an Assistant
// interaction.
//
// It records messages exchanged between the user, Assistant, LLM,
// tool calls, and tool results.
//
// A conversation may look like:
//
//   User Message
//      ↓
//   Assistant -> Tool Call
//      ↓
//   Tool Result
//      ↓
//   Assistant -> Another Tool Call
//      ↓
//   Tool Result
//      ↓
//   Final Assistant Message
//
// The conversation exists so the LLM can understand what has already
// happened and what information has already been retrieved.
//
// It does NOT store business truth.
// Live business state must come from Host Project tools.
