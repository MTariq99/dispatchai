package policy

// This file defines the generic policy evaluation contract.
//
// A policy answers:
//
//   "Is this requested operation allowed?"
//
// Policies are deterministic application rules and are not delegated
// to the LLM.
//
// The LLM can request an action, but application policy determines
// whether that action can proceed.
