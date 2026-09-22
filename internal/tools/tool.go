package tools

// This file defines the generic representation of a tool capability.
//
// IMPORTANT:
//
// A Tool in DispatchAI is a description of a capability provided by
// the Host Project.
//
// It is NOT necessarily the implementation of that capability.
//
// For example:
//
//   Name:
//       find_delayed_orders
//
//   Description:
//       Find orders delayed by a specified number of minutes.
//
// The Host Project owns the actual implementation.
//
// DispatchAI only needs enough information to let the LLM understand
// what capability is available.
