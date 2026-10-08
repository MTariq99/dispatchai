package policy

// Outcome represents what the Policy Engine decided about a tool call.
type Outcome string

const (
	Allow           Outcome = "ALLOW"
	Deny            Outcome = "DENY"
	RequireApproval Outcome = "REQUIRE_APPROVAL"
)

// Decision is the result of evaluating a tool call against all rules.
type Decision struct {
	Outcome Outcome
	Reason  string // human/LLM-readable explanation, especially for Deny/RequireApproval
}

func AllowDecision() Decision {
	return Decision{Outcome: Allow}
}

func DenyDecision(reason string) Decision {
	return Decision{Outcome: Deny, Reason: reason}
}

func RequireApprovalDecision(reason string) Decision {
	return Decision{Outcome: RequireApproval, Reason: reason}
}
