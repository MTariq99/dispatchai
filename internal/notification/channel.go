package notification

// This file defines the generic notification channel contract.
//
// Possible channels include:
//
//   - email
//   - SMS
//   - push
//   - webhook
//   - WhatsApp
//   - Slack
//   - Teams
//
// A channel describes how a notification can be delivered.
//
// Concrete business integrations should remain behind adapters or,
// under the locked architecture, inside the Host Project.
