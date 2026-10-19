package utils

import (
	"fmt"

	"github.com/google/uuid"
)

func ParseUUID(strVal, label string) (uuid.UUID, error) {
	if strVal == "" {
		return uuid.Nil, fmt.Errorf("%s is required", label)
	}
	id, err := uuid.Parse(strVal)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s: must be a valid UUID", label)
	}
	return id, nil
}
