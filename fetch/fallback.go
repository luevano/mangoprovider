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

// ErrChallenged: Cloudflare challenged the request and no headless browser got past it
// (FlareSolverr isn't enabled, rod's Chromium failed or was challenged too). It also matches
// mango.ErrSiteDown, so mirrors move on.
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

// challenged: Whether resp is a challenge; resp's body can still be read.
func challenged(resp *http.Response) (bool, error) {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusServiceUnavailable {
		return false, nil
	}
	body, err := mango.ReadBody(resp.Body)
	resp.Body.Close()
	if err != nil {
		return false, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return Challenged(resp.StatusCode, resp.Header, body), nil
}

// fallback: Requests go through next, a challenged one is retried in a headless browser:
// FlareSolverr if enabled (session made on the first challenge), else rod's (started on the
// first challenge). Both are ended by Close.
type fallback struct {
	next       http.RoundTripper
	headless   mango.HeadlessOptions
	newBrowser func() Browser
	log        *mango.Logger

	mu      sync.Mutex
	solver  *flaresolverr.TransportFlaresolverr
	browser Browser
	// challenged, warned: hosts already reported as challenged, as fetched through the fallback
	challenged, warned map[string]bool
}

func newFallback(next http.RoundTripper, headless mango.HeadlessOptions, newBrowser func() Browser, log *mango.Logger) *fallback {
	return &fallback{next: next, headless: headless, newBrowser: newBrowser, log: log, challenged: map[string]bool{}, warned: map[string]bool{}}
}

func (f *fallback) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := f.next.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	if ok, err := challenged(resp); err != nil || !ok {
		return resp, err
	}

	host := mango.Host(r.URL.String())
	via, retry := "the headless browser", f.viaBrowser
	if f.headless.UseFlaresolverr && strings.TrimSpace(f.headless.FlaresolverrURL) != "" {
		via, retry = "FlareSolverr", f.viaFlaresolverr
	}
	if resp, err = retry(r, f.once(f.challenged, host)); err != nil {
		return nil, err
	}
	// a site that needs it often is better read with the Headless strategy from the start
	format, a := "challenged at %s, fetched through %s, consider fetch.Headless for this site", []any{host, via}
	if f.once(f.warned, host) {
		f.log.Status(r.Context(), format, a...)
	} else {
		f.log.Log(format, a...)
	}
	return resp, nil
}

// once: Whether host is new to seen, now added.
func (f *fallback) once(seen map[string]bool, host string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	first := !seen[host]
	seen[host] = true
	return first
}

// status reports what the request now waits on: a status the first time for host, a minor
// one after.
func (f *fallback) status(ctx context.Context, first bool, format string, a ...any) {
	if first {
		f.log.Status(ctx, format, a...)
		return
	}
	f.log.Progress(ctx, 0, 0, format, a...)
}

func (f *fallback) viaFlaresolverr(r *http.Request, first bool) (*http.Response, error) {
	f.mu.Lock()
	if f.solver == nil {
		f.solver = flaresolverr.NewTransportClient(f.headless.FlaresolverrURL, f.headless.FlaresolverrClient)
	}
	solver := f.solver
	f.mu.Unlock()

	f.status(r.Context(), first, "cloudflare challenge on %s, solving with flaresolverr (up to %ds)", mango.Host(r.URL.String()), int(SolveTimeout.Seconds()))
	ctx, cancel := context.WithTimeout(r.Context(), SolveTimeout)
	retry, err := clone(ctx, r)
	if err != nil {
		cancel()
		return nil, err
	}
	resp, err := solver.RoundTrip(retry)
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

func (f *fallback) viaBrowser(r *http.Request, first bool) (*http.Response, error) {
	f.mu.Lock()
	if f.browser == nil {
		f.browser = f.newBrowser()
	}
	browser := f.browser
	f.mu.Unlock()

	f.status(r.Context(), first, "cloudflare challenge on %s, trying the headless browser", mango.Host(r.URL.String()))
	retry, err := clone(r.Context(), r)
	if err != nil {
		return nil, err
	}
	resp, err := browser.RoundTrip(retry)
	if err != nil {
		if r.Context().Err() != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %w (headless browser: %w)", mango.ErrSiteDown, ErrChallenged, err)
	}
	ok, err := challenged(resp)
	if err != nil {
		return nil, err
	}
	if ok {
		resp.Body.Close()
		return nil, fmt.Errorf("%w: %w (the headless browser was challenged too)", mango.ErrSiteDown, ErrChallenged)
	}
	return resp, nil
}

// clone: r with ctx, its body again.
func clone(ctx context.Context, r *http.Request) (*http.Request, error) {
	retry := r.Clone(ctx)
	if r.Body != nil && r.Body != http.NoBody {
		if r.GetBody == nil {
			return nil, fmt.Errorf("%w: can't resend the request body", mango.ErrSiteDown)
		}
		var err error
		if retry.Body, err = r.GetBody(); err != nil {
			return nil, err
		}
	}
	return retry, nil
}

// Close destroys the FlareSolverr session and closes the browser, if started.
func (f *fallback) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	var errs []error
	if f.solver != nil {
		errs = append(errs, f.solver.Close())
		f.solver = nil
	}
	if f.browser != nil {
		errs = append(errs, f.browser.Close())
		f.browser = nil
	}
	return errors.Join(errs...)
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
