package utils

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

func ParseStrToInt64(s string) (int64, error) {
	Id64, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ID format: %w", err)
	}
	return Id64, nil
}

func ParseInt64(strVal, label string) (int64, error) {
	if strVal == "" {
		return 0, fmt.Errorf("%s is required", label)
	}
	parsed, err := strconv.ParseInt(strVal, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be a number", label)
	}
	return parsed, nil
}

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
