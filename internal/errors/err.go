package errs

import (
	"fmt"
)

func Wrap(operation error, cause error) error {
	if cause == nil {
		return nil
	}
	if operation == nil {
		return cause
	}

	return fmt.Errorf("%w: %w", operation, cause)
}

func Wrapf(operation error, cause error, detail ...any) error {
	if cause == nil {
		return nil
	}
	if operation == nil {
		return cause
	}

	if len(detail) == 0 {
		return fmt.Errorf("%w: %w", operation, cause)
	}

	format, ok := detail[0].(string)
	if !ok {
		return fmt.Errorf("%w: %w", operation, cause)
	}

	return fmt.Errorf(
		"%w: %s: %w",
		operation,
		fmt.Sprintf(format, detail[1:]...),
		cause,
	)
}
