package fetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/flaresolverr"
)

// ErrChallenged: Cloudflare challenged the request and FlareSolverr isn't enabled. It also
// matches mango.ErrSiteDown, so mirrors move on.
var ErrChallenged = errors.New("cloudflare challenge, enable flaresolverr (providers.headless.use_flaresolverr)")

// SolveTimeout: How long FlareSolverr gets for a challenge. Solving takes 10-20s; with
// mirrors, an unsolvable one shouldn't hold the next for long.
var SolveTimeout = 60 * time.Second

// Challenged: The response is a Cloudflare challenge ("Just a moment..."), body is its body.
func Challenged(status int, header http.Header, body []byte) bool {
	if status != http.StatusForbidden && status != http.StatusServiceUnavailable {
		return false
	}
	return header.Get("Cf-Mitigated") == "challenge" ||
		bytes.Contains(body, []byte("<title>Just a moment...</title>"))
}

// cloudflare: Requests go through next, a challenged one is retried through FlareSolverr.
// The session is made on the first challenge and destroyed by Close.
type cloudflare struct {
	next     http.RoundTripper
	headless mango.HeadlessOptions
	log      *mango.Logger

	mu     sync.Mutex
	solver *flaresolverr.TransportFlaresolverr
}

func newCloudflare(next http.RoundTripper, headless mango.HeadlessOptions, log *mango.Logger) *cloudflare {
	return &cloudflare{next: next, headless: headless, log: log}
}

func (c *cloudflare) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := c.next.RoundTrip(r)
	if err != nil || (resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusServiceUnavailable) {
		return resp, err
	}
	body, err := mango.ReadBody(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if !Challenged(resp.StatusCode, resp.Header, body) {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp, nil
	}

	solver, err := c.flaresolverr()
	if err != nil {
		return nil, err
	}
	c.log.Status(r.Context(), "cloudflare challenge on %s, solving with flaresolverr (up to %ds)", mango.Host(r.URL.String()), int(SolveTimeout.Seconds()))
	ctx, cancel := context.WithTimeout(r.Context(), SolveTimeout)
	retry := r.Clone(ctx)
	if r.Body != nil && r.Body != http.NoBody {
		if r.GetBody == nil {
			cancel()
			return nil, fmt.Errorf("%w: can't resend the request body", mango.ErrSiteDown)
		}
		if retry.Body, err = r.GetBody(); err != nil {
			cancel()
			return nil, err
		}
	}
	resp, err = solver.RoundTrip(retry)
	if err != nil {
		cancel()
		if r.Context().Err() != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %w", mango.ErrSiteDown, err)
	}
	resp.Body = &cancelBody{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

func (c *cloudflare) flaresolverr() (*flaresolverr.TransportFlaresolverr, error) {
	if !c.headless.UseFlaresolverr || strings.TrimSpace(c.headless.FlaresolverrURL) == "" {
		return nil, fmt.Errorf("%w: %w", mango.ErrSiteDown, ErrChallenged)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.solver == nil {
		c.solver = flaresolverr.NewTransport(c.headless.FlaresolverrURL)
	}
	return c.solver, nil
}

// Close destroys the FlareSolverr session, if there's one.
func (c *cloudflare) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.solver == nil {
		return nil
	}
	err := c.solver.Close()
	c.solver = nil
	return err
}

// cancelBody: The retry's timeout lasts until its body is closed.
type cancelBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelBody) Close() error {
	defer b.cancel()
	return b.ReadCloser.Close()
}
