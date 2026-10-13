package mangoprovider

import (
	"errors"

	"github.com/luevano/libmangal/netutil"
	"github.com/luevano/mangodex"
	"github.com/luevano/mangoplus"
	"github.com/luevano/mangoplus/creators"
)

// Errors to check with errors.Is, ErrNotFound and ErrRateLimited are the same values as
// libmangal's.
var (
	ErrNotFound     = netutil.ErrNotFound
	ErrNotSupported = errors.New("not supported")
	ErrRateLimited  = netutil.ErrRateLimited
	// ErrSiteDown: the website didn't answer (connection error, 5xx, a challenge that couldn't be
	// solved), another mirror may.
	ErrSiteDown = errors.New("site down")
)

// apiError: An error of an api client (mangodex, mangoplus) that also matches the shared
// sentinel it stands for, the clients don't depend on libmangal.
type apiError struct {
	err    error
	target error
}

func (e *apiError) Error() string        { return e.err.Error() }
func (e *apiError) Unwrap() error        { return e.err }
func (e *apiError) Is(target error) bool { return target == e.target }

// shared: err, made to match ErrNotFound/ErrRateLimited if it's an api client's not found or
// rate limit.
func shared(err error) error {
	switch {
	case err == nil, errors.Is(err, ErrNotFound), errors.Is(err, ErrRateLimited):
		return err
	case errors.Is(err, mangodex.ErrNotFound), errors.Is(err, mangoplus.ErrNotFound), errors.Is(err, creators.ErrNotFound):
		return &apiError{err: err, target: ErrNotFound}
	case errors.Is(err, mangodex.ErrRateLimited), errors.Is(err, mangoplus.ErrRateLimited):
		return &apiError{err: err, target: ErrRateLimited}
	}
	return err
}
