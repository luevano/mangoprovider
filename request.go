package mangoprovider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// maxBodySize caps what is read from a site: a page, an api response or an image.
var maxBodySize int64 = 100 << 20

// ReadBody reads r, an error if it's over 100 MiB.
func ReadBody(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxBodySize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBodySize {
		return nil, fmt.Errorf("response over %d bytes", maxBodySize)
	}
	return body, nil
}

// RateLimitError: The site answered 429 too many requests.
type RateLimitError struct {
	// RetryAfter: How long the site asked to wait, 0 if it didn't say.
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("429 too many requests (retry after %s)", e.RetryAfter)
	}
	return "429 too many requests"
}

func (e *RateLimitError) Is(target error) bool {
	return target == ErrRateLimited
}

// RetryAfterDuration: How long to wait before trying again, for callers that don't import
// mangoprovider.
func (e *RateLimitError) RetryAfterDuration() time.Duration {
	return e.RetryAfter
}

// StatusError is the error for a response status, nil for 200: 404 matches ErrNotFound, 429 is
// a *RateLimitError and 5xx matches ErrSiteDown.
func StatusError(status int, header http.Header) error {
	text := strings.ToLower(http.StatusText(status))
	switch {
	case status == http.StatusOK:
		return nil
	case status == http.StatusNotFound:
		return fmt.Errorf("%d %w", status, ErrNotFound)
	case status == http.StatusTooManyRequests:
		return &RateLimitError{RetryAfter: retryAfter(header.Get("Retry-After"))}
	case status >= 500:
		return fmt.Errorf("%w: %d %s", ErrSiteDown, status, text)
	}
	return fmt.Errorf("%d %s", status, text)
}

// Fetch gets url with the user agent, once limiter (if any) allows it, whatever the status.
// The body is read with ReadBody.
func Fetch(ctx context.Context, client *http.Client, limiter *rate.Limiter, userAgent, url string) ([]byte, *http.Response, error) {
	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			return nil, nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := ReadBody(resp.Body)
	return body, resp, err
}
