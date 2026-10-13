package mangoprovider

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStatusError(t *testing.T) {
	if err := StatusError(http.StatusOK, nil); err != nil {
		t.Errorf("200: %v", err)
	}
	if err := StatusError(http.StatusNotFound, nil); !errors.Is(err, ErrNotFound) || err.Error() != "404 not found" {
		t.Errorf("404: %v", err)
	}
	if err := StatusError(http.StatusBadGateway, nil); !errors.Is(err, ErrSiteDown) {
		t.Errorf("502: %v", err)
	}
	if err := StatusError(http.StatusForbidden, nil); errors.Is(err, ErrSiteDown) || err.Error() != "403 forbidden" {
		t.Errorf("403: %v", err)
	}
	err := StatusError(http.StatusTooManyRequests, http.Header{"Retry-After": {"7"}})
	var rateLimit *RateLimitError
	if !errors.Is(err, ErrRateLimited) || !errors.As(err, &rateLimit) || rateLimit.RetryAfterDuration() != 7*time.Second {
		t.Errorf("429: %v", err)
	}
}

func TestReadBodyCapped(t *testing.T) {
	defer func(m int64) { maxBodySize = m }(maxBodySize)
	maxBodySize = 4

	if body, err := ReadBody(strings.NewReader("1234")); err != nil || string(body) != "1234" {
		t.Errorf("at the cap: %q, %v", body, err)
	}
	if _, err := ReadBody(strings.NewReader("12345")); err == nil {
		t.Error("over the cap: wanted an error")
	}
}
