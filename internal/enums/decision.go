package enums

type Outcome string

const (
	Allow           Outcome = "ALLOW"
	Deny            Outcome = "DENY"
	RequireApproval Outcome = "REQUIRE_APPROVAL"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
)
