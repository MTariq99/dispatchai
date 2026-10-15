package enums

import (
	"regexp"
	"time"
)

const (
	MaxOnboardingBodyBytes = 1 << 20  // 1 MiB
	MaxEnrollmentBodyBytes = 64 << 10 // 64 KiB
	EnrollmentTokenTTL     = 15 * time.Minute
	AdminActorKey          = "admin_actor"
)

var (
	EnrollmentTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
	KeyIDPattern           = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)
