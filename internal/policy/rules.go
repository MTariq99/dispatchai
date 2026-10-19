package policy

import "github.com/mtariq99/dispatchai/models"

type Rule func(call *models.Call, execCtx *models.ExecutionContext, callCounts map[string]int) Decision

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
