package idempotency

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

// This file defines the idempotency contract used to prevent duplicate
// execution.
//
// This is especially important for operations such as notifications.
//
// Example:
//
//   Request:
//       send_notification(id=123)
//
//   Network timeout occurs.
//
//   Assistant retries.
//
// Without idempotency:
//
//   Notification 1 -> sent
//   Notification 2 -> sent again
//
// With idempotency:
//
//   Existing execution detected
//        ↓
//   Duplicate execution prevented
//
// The actual business operation remains Host Project-owned.

func (is *ToolIdempotencyStore) BuildKey(conversationId uuid.UUID, call *models.Call) string {
	raw := conversationId.String() + "|" + call.Name + "|" + string(call.Arguments)

	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
