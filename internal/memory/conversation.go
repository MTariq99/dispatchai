package memory

// This file provides the application-level conversation memory behavior.
//
// It is responsible for:
//
//   - loading conversation history
//   - appending messages
//   - saving conversation state
//   - deleting conversation state
//
// This is Assistant memory, not business state.
//
// For example, this file may remember that the user previously asked
// about delayed orders, but the current order status must still come
// from the Host Project.
