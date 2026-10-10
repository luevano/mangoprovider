package mangoprovider

import "errors"

// Errors to check with errors.Is.
var (
	ErrNotFound     = errors.New("not found")
	ErrNotSupported = errors.New("not supported")
)
