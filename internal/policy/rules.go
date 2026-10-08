package policy

import "github.com/mtariq99/dispatchai/models"

// Rule evaluates one tool call and returns a Decision. callCounts maps
// tool name -> number of times it has already succeeded in this run,
// letting rules reason about conversation history without needing the
// full Conversation type (keeps this package decoupled from assistant).
type Rule func(call *models.Call, execCtx *models.ExecutionContext, callCounts map[string]int) Decision

// MaxNotificationsPerRun denies send_notification once it has already
// succeeded this many times in the current run.
func MaxNotificationsPerRun(limit int) Rule {
	return func(call *models.Call, execCtx *models.ExecutionContext, callCounts map[string]int) Decision {
		if call.Name != "send_notification" {
			return AllowDecision()
		}

		count := callCounts["send_notification"]
		if count >= limit {
			return DenyDecision("notification limit reached for this conversation")
		}

		return AllowDecision()
	}
}
