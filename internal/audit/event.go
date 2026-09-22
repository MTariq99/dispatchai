package audit

// This file defines the structure of an audit event.
//
// Events may contain:
//
//   - tenant
//   - user
//   - request ID
//   - conversation ID
//   - action/tool
//   - arguments
//   - result
//   - policy decision
//   - timestamp
//
// Sensitive information must be handled carefully and should not be
// logged blindly.
