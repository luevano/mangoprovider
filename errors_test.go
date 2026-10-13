package mangoprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/mangodex"
	"github.com/luevano/mangoplus"
	"github.com/luevano/mangoplus/creators"
)

func TestSharedErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
	}{
		{"mangodex 404", &mangodex.HTTPError{Method: "GET", Path: "/manga/x", StatusCode: http.StatusNotFound}, libmangal.ErrNotFound},
		{"mangodex 429", &mangodex.RateLimitError{Method: "GET", Path: "/manga", RetryAfter: 7 * time.Second}, libmangal.ErrRateLimited},
		{"mangoplus 404", &mangoplus.HTTPError{Method: "GET", Path: "/title_detailV3", StatusCode: http.StatusNotFound}, libmangal.ErrNotFound},
		{"mangoplus 429", &mangoplus.HTTPError{Method: "GET", Path: "/title_list", StatusCode: http.StatusTooManyRequests}, libmangal.ErrRateLimited},
		{"mangoplus 10522", &mangoplus.ServerError{Method: "GET", Path: "/title_list", Code: 10522}, libmangal.ErrRateLimited},
		{"mangoplus 11103", &mangoplus.ServerError{Method: "GET", Path: "/manga_viewer", Code: 11103}, libmangal.ErrNotFound},
		{"creators 404", &creators.HTTPError{Method: "GET", Path: "/titles", StatusCode: http.StatusNotFound}, libmangal.ErrNotFound},
		{"mango not found", fmt.Errorf("404 %w", ErrNotFound), libmangal.ErrNotFound},
		{"mango 429", &RateLimitError{}, libmangal.ErrRateLimited},
	}
	for _, tt := range tests {
		p := &Provider{
			ProviderInfo: libmangal.ProviderInfo{ID: "mango-test"},
			F: Functions{
				SearchMangas: func(context.Context, Store, string) ([]mangadata.Manga, error) {
					return nil, fmt.Errorf("search: %w", tt.err)
				},
			},
		}
		_, err := p.SearchMangas(context.Background(), "x")
		if !errors.Is(err, tt.target) || !errors.Is(err, tt.err) {
			t.Errorf("%s: %v should match %v and keep the original", tt.name, err, tt.target)
		}
		other := libmangal.ErrNotFound
		if tt.target == other {
			other = libmangal.ErrRateLimited
		}
		if errors.Is(err, other) {
			t.Errorf("%s: %v shouldn't match %v", tt.name, err, other)
		}
		if want := "mango-test: search \"x\": search: " + tt.err.Error(); err.Error() != want {
			t.Errorf("%s: %q, want %q", tt.name, err, want)
		}
	}

	_, err := (&Provider{F: Functions{SearchMangas: func(context.Context, Store, string) ([]mangadata.Manga, error) {
		return nil, &mangodex.RateLimitError{RetryAfter: 7 * time.Second}
	}}}).SearchMangas(context.Background(), "x")
	var retrier interface{ RetryAfterDuration() time.Duration }
	if !errors.As(err, &retrier) || retrier.RetryAfterDuration() != 7*time.Second {
		t.Errorf("%v: lost the mangodex RetryAfterDuration", err)
	}

	if err := shared(nil); err != nil {
		t.Errorf("nil: %v", err)
	}
	if err := shared(&mangodex.HTTPError{StatusCode: http.StatusInternalServerError}); errors.Is(err, libmangal.ErrNotFound) || errors.Is(err, libmangal.ErrRateLimited) {
		t.Errorf("500: %v shouldn't match the shared sentinels", err)
	}
}
