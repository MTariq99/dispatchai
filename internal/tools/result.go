package tools

// This file represents the result returned after the Host Project
// executes a requested tool.
//
// A result may contain:
//
//   - success/failure
//   - structured data
//   - error information
//   - metadata
//
// The result is then converted into information the LLM can reason
// about.
//
// The result is authoritative for the operation it represents because
// it comes from the Host Project.
