// Package fetch builds the http client a site is read with, like keiyoushi's per-source
// OkHttp clients: the shared client plus RoundTripper middlewares picked per site (rate limit,
// headers, headless browser fallback, headless browser).
package fetch

import (
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"

	"github.com/luevano/libmangal/netutil"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/time/rate"
)

// Strategy: How a site's requests are made.
type Strategy int

const (
	// Direct: plain requests.
	Direct Strategy = iota
	// HeadlessFallback: direct, a request Cloudflare challenges is retried in a headless
	// browser: FlareSolverr if mango.HeadlessOptions enable it (its session keeps the
	// clearance), else rod's Chromium (see Headless). Each fallback is logged, the first per
	// host as a status too: a site that needs it often should use Headless.
	HeadlessFallback
	// Headless: every request loads the page in rod's headless Chromium (sites that build the
	// html with scripts), downloaded on first use. The body is the page's html once loaded.
	Headless
)

// Browser loads the pages in a headless browser, see rod.TransportRod.
type Browser interface {
	http.RoundTripper
	Close() error
}

// Middleware wraps the transport of a request, like an OkHttp interceptor.
type Middleware func(next http.RoundTripper) http.RoundTripper

// RoundTripperFunc is a func as an http.RoundTripper.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Config: How a site is fetched. The zero value is direct requests.
type Config struct {
	Strategy Strategy
	// Limiter: Optional, every request (redirects and retries through FlareSolverr included)
	// waits for it.
	Limiter *rate.Limiter
	// Headless: How the Headless strategy (and HeadlessFallback's browser) loads the pages.
	Headless rod.Options
	// NewBrowser: Optional, the browser of Headless and HeadlessFallback (made on first use),
	// rod.NewTransport with Headless by default.
	NewBrowser func() Browser
	// Middlewares: The site's own hooks (headers, user agent, url rewrites, ...), the first is
	// the outermost. They run before the retries, the limiter and the strategy.
	Middlewares []Middleware
	// Retry: Optional, transient failures (429, 5xx) are tried again (see Retries), each try
	// waiting for the limiter.
	Retry *Retry
	// Debug: Optional, logs the headless browser's page loads (mango.Options.Debug).
	Debug func(format string, a ...any)
}

// Client is a site's http client, Close it with the provider.
type Client struct {
	*http.Client
	direct  *http.Client
	closers []func() error
}

// NewClient: A copy of base (nil is mango.DefaultOptions' client, the shared one stays as is)
// with its own cookie jar (unless it has one) and the transport config asks for.
func NewClient(base *http.Client, headless mango.HeadlessOptions, log *mango.Logger, config Config) (*Client, error) {
	if base == nil {
		base = mango.DefaultOptions().HTTPClient
	}
	direct := *base
	// some sites set a session cookie on a redirect or the first page
	if direct.Jar == nil {
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, err
		}
		direct.Jar = jar
	}
	transport := direct.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	newBrowser := config.NewBrowser
	if newBrowser == nil {
		newBrowser = func() Browser { return rod.NewTransport(config.Headless, log) }
	}
	if debug := config.Debug; debug != nil {
		plain := newBrowser
		newBrowser = func() Browser { return loggedBrowser(plain(), debug) }
	}

	c := &Client{direct: &direct}
	switch config.Strategy {
	case Direct:
	case HeadlessFallback:
		f := newFallback(transport, headless, newBrowser, log)
		c.closers = append(c.closers, f.Close)
		transport = f
	case Headless:
		browser := newBrowser()
		c.closers = append(c.closers, browser.Close)
		transport = browser
	default:
		return nil, errors.New("fetch: unknown strategy")
	}
	if config.Limiter != nil {
		transport = RateLimit(config.Limiter)(transport)
	}
	if config.Retry != nil {
		transport = Retries(*config.Retry, log)(transport)
	}
	for i := len(config.Middlewares) - 1; i >= 0; i-- {
		transport = config.Middlewares[i](transport)
	}

	client := direct
	client.Transport = transport
	c.Client = &client
	return c, nil
}

// Direct: The client without the strategy and middlewares (same cookies), for requests that
// shouldn't start a browser or FlareSolverr.
func (c *Client) Direct() *http.Client {
	return c.direct
}

// Close ends what the strategy started (FlareSolverr session, browser).
func (c *Client) Close() error {
	var errs []error
	for _, close := range c.closers {
		errs = append(errs, close())
	}
	return errors.Join(errs...)
}

// loggedBrowser: b logging each page load like netutil.LoggingTransport, "headless" first.
func loggedBrowser(b Browser, debug func(format string, a ...any)) Browser {
	logf := func(format string, a ...any) { debug("headless "+format, a...) }
	return struct {
		http.RoundTripper
		io.Closer
	}{netutil.NewLoggingTransport(b, logf, nil), b}
}

// RateLimit: Every request waits for l (or its context).
func RateLimit(l *rate.Limiter) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if err := l.Wait(r.Context()); err != nil {
				return nil, err
			}
			return next.RoundTrip(r)
		})
	}
}

// Headers: Sets the headers a request doesn't have yet, like keiyoushi's headersBuilder.
func Headers(headers map[string]string) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			var clone *http.Request
			for k, v := range headers {
				if r.Header.Get(k) != "" {
					continue
				}
				// a RoundTripper mustn't change the caller's request
				if clone == nil {
					clone = r.Clone(r.Context())
				}
				clone.Header.Set(k, v)
			}
			if clone == nil {
				return next.RoundTrip(r)
			}
			return next.RoundTrip(clone)
		})
	}
}
