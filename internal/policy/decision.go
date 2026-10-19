package policy

type Outcome string

const (
	Allow           Outcome = "ALLOW"
	Deny            Outcome = "DENY"
	RequireApproval Outcome = "REQUIRE_APPROVAL"
)

type Decision struct {
	Outcome Outcome
	Reason  string
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
