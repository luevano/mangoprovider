package mangoprovider

import "errors"

// Errors to check with errors.Is.
var (
	ErrNotFound     = errors.New("not found")
	ErrNotSupported = errors.New("not supported")
	// ErrSiteDown: the website didn't answer (connection error, 5xx, a challenge that couldn't be
	// solved), another mirror may.
	ErrSiteDown = errors.New("site down")
)
