package errs

import "errors"

var ErrToolCallConflict = errors.New("tool call ID reused with different tool name or arguments")
