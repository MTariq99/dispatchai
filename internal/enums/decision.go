package enums

type Outcome string

const (
	Allow           Outcome = "ALLOW"
	Deny            Outcome = "DENY"
	RequireApproval Outcome = "REQUIRE_APPROVAL"
)
