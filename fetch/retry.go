package fetch

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v7"
	"github.com/luevano/libmangal/netutil"
	mango "github.com/luevano/mangoprovider"
	"golang.org/x/time/rate"
)

// Retry: How a request that failed for a while (429, 5xx) is tried again: exponential backoff
// (with jitter) or the Retry-After the site asks for. The zero value is DefaultRetry's.
type Retry struct {
	// MaxTries: Every attempt, the first included; 1 never retries.
	MaxTries uint
	// MaxElapsed: No retry that would end past this since the first attempt, e.g. a longer
	// Retry-After: the response is returned as is (see mango.RateLimitError).
	MaxElapsed time.Duration
	// InitialInterval: The first wait when the site doesn't say, then doubled up to MaxInterval.
	InitialInterval time.Duration
	MaxInterval     time.Duration
	// Statuses: Optional, which statuses are tried again, Retryable by default. APIs whose 429
	// mustn't be retried soon (a ban or lockout for repeating it) pass ServerErrors.
	Statuses func(status int) bool
	// Skip: Optional, requests never tried again (e.g. best-effort reports).
	Skip func(req *http.Request) bool
}

// DefaultRetry: Small enough not to hold a search for long: 3 tries within 30s, waiting ~1s
// then ~2s.
func DefaultRetry() Retry {
	return Retry{MaxTries: 3, MaxElapsed: 30 * time.Second, InitialInterval: time.Second, MaxInterval: 10 * time.Second}
}

func (r Retry) withDefaults() Retry {
	d := DefaultRetry()
	if r.MaxTries == 0 {
		r.MaxTries = d.MaxTries
	}
	if r.MaxElapsed <= 0 {
		r.MaxElapsed = d.MaxElapsed
	}
	if r.InitialInterval <= 0 {
		r.InitialInterval = d.InitialInterval
	}
	if r.MaxInterval <= 0 {
		r.MaxInterval = d.MaxInterval
	}
	if r.Statuses == nil {
		r.Statuses = Retryable
	}
	return r
}

// Retryable: A status worth trying again soon: 429 and 5xx, except 501 (not implemented).
func Retryable(status int) bool {
	return status == http.StatusTooManyRequests || (status >= 500 && status != http.StatusNotImplemented)
}

// ServerErrors: Retryable without the 429.
func ServerErrors(status int) bool {
	return status != http.StatusTooManyRequests && Retryable(status)
}

// RetryClient: A copy of base (nil is mango.DefaultOptions' client) whose requests are retried
// as r says, every try waiting for limiter if set. For API wrappers that take an *http.Client
// (a limiter of their own only covers the first try).
func RetryClient(base *http.Client, r Retry, limiter *rate.Limiter, log *mango.Logger) *http.Client {
	if base == nil {
		base = mango.DefaultOptions().HTTPClient
	}
	client := *base
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	if limiter != nil {
		transport = RateLimit(limiter)(transport)
	}
	client.Transport = Retries(r, log)(transport)
	return &client
}

// retryableError: The attempt answered a Retryable status.
type retryableError struct {
	status int
}

func (e *retryableError) Error() string {
	// Cloudflare's 52x have no text
	return strings.TrimSpace(fmt.Sprintf("%d %s", e.status, strings.ToLower(http.StatusText(e.status))))
}

// Retries: Requests answered with a Retryable status are tried again as r says, each wait
// reported as a status on log. A Cloudflare challenge isn't retried (see HeadlessFallback),
// nor is a request whose body can't be sent again. Once out of tries the last response is
// returned.
func Retries(r Retry, log *mango.Logger) Middleware {
	r = r.withDefaults()
	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if r.MaxTries <= 1 || (r.Skip != nil && r.Skip(req)) {
				return next.RoundTrip(req)
			}
			ctx := req.Context()
			exp := backoff.NewExponentialBackOff()
			exp.InitialInterval, exp.MaxInterval, exp.Multiplier = r.InitialInterval, r.MaxInterval, 2

			var last *http.Response
			tries := 0
			resp, err := backoff.Retry(ctx, func() (*http.Response, error) {
				attempt := req
				if tries++; tries > 1 {
					var err error
					if attempt, err = clone(ctx, req); err != nil {
						return nil, backoff.Permanent(err)
					}
				}
				resp, err := next.RoundTrip(attempt)
				if err != nil {
					return nil, backoff.Permanent(err)
				}
				if !r.Statuses(resp.StatusCode) {
					return resp, nil
				}
				if ok, err := challenged(resp); err != nil || ok {
					return resp, backoff.Permanent(errNotRetried)
				}
				// read for the caller if it's the last, else closed for the next try
				body, err := mango.ReadBody(resp.Body)
				resp.Body.Close()
				if err != nil {
					return nil, backoff.Permanent(err)
				}
				resp.Body = io.NopCloser(bytes.NewReader(body))
				last = resp
				failed := &retryableError{status: resp.StatusCode}
				if after := netutil.RetryAfter(resp.Header.Get("Retry-After")); after > 0 {
					return resp, backoff.RetryAfter(after, failed)
				}
				return resp, failed
			},
				backoff.WithBackOff(exp),
				backoff.WithMaxTries(r.MaxTries),
				backoff.WithMaxElapsedTime(r.MaxElapsed),
				backoff.WithNotify(func(err error, wait time.Duration) {
					// the status, without the RetryAfterError's own text (the wait is shown)
					if failed, ok := errors.AsType[*retryableError](err); ok {
						err = failed
					}
					log.Status(ctx, "%s: %s, retry %d/%d in %s", mango.Host(req.URL.String()), err, tries, r.MaxTries-1, wait.Round(100*time.Millisecond))
				}),
			)
			retryErr := backoff.AsRetryError(err)
			switch {
			case err == nil:
				return resp, nil
			case errors.Is(err, errNotRetried):
				return resp, nil
			case retryErr == nil:
				return nil, err
			case errors.Is(retryErr.Cause, backoff.ErrPermanent):
				return nil, retryErr.LastErr
			case errors.Is(retryErr.Cause, backoff.ErrExhausted), errors.Is(retryErr.Cause, backoff.ErrMaxElapsedTime):
				if last != nil {
					return last, nil
				}
				return nil, retryErr.LastErr
			}
			// the ctx ended
			return nil, retryErr.Cause
		})
	}
}

// errNotRetried: The response is returned as is.
var errNotRetried = errors.New("not retried")
