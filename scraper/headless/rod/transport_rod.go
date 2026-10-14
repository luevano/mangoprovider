package rod

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	mango "github.com/luevano/mangoprovider"
)

var _ http.RoundTripper = (*TransportRod)(nil)

// Options: How the pages are loaded.
type Options struct {
	// LoadWait: How long to wait for the page to load, instead of its load event (e.g. html
	// filled in by scripts after it).
	LoadWait time.Duration
	// LocalStorage: Values set in the site's local storage before the request.
	LocalStorage map[string]string
}

// TransportRod is an http.RoundTripper that loads the pages in a headless browser.
type TransportRod struct {
	// launched on the first request, so errors can be returned instead of panicking
	browser    *rod.Browser
	launchErr  error
	launchOnce sync.Once

	loadWait     time.Duration
	localStorage map[string]string
	log          *mango.Logger
}

func (t *TransportRod) Close() error {
	if t.browser == nil {
		return nil
	}
	return t.browser.Close()
}

// NewTransport creates a new transport with the browser setup.
func NewTransport(options Options, log *mango.Logger) *TransportRod {
	return &TransportRod{
		loadWait:     options.LoadWait,
		localStorage: options.LocalStorage,
		log:          log,
	}
}

// launch: Starts the browser once, on the first request.
func (t *TransportRod) launch(ctx context.Context) (*rod.Browser, error) {
	t.launchOnce.Do(func() {
		l := launcher.New().Leakless(runtime.GOOS == "linux")
		if l.Get(flags.Bin) == "" {
			// rod's own download logs to stdout, which must stay clean (e.g. --json)
			b := launcher.NewBrowser()
			b.Logger = logger{t.log}
			if b.Validate() != nil {
				t.log.Status(ctx, "downloading chromium for the headless browser")
			}
			bin, err := b.Get()
			if err != nil {
				t.launchErr = fmt.Errorf("download browser: %w", err)
				return
			}
			l = l.Bin(bin)
		}
		t.log.Status(ctx, "starting the headless browser")
		u, err := l.Launch()
		if err != nil {
			t.launchErr = fmt.Errorf("launch browser: %w", err)
			return
		}
		browser := rod.New().ControlURL(u)
		if err := browser.Connect(); err != nil {
			t.launchErr = fmt.Errorf("connect to browser: %w", err)
			return
		}
		t.browser = browser
	})
	return t.browser, t.launchErr
}

// RoundTrip loads the page in the browser, its html is the body.
func (t *TransportRod) RoundTrip(r *http.Request) (*http.Response, error) {
	browser, err := t.launch(r.Context())
	if err != nil {
		return nil, err
	}

	// Only use the base URL when there are local storage values to set
	pageURL := ""
	if len(t.localStorage) != 0 {
		baseURL, err := url.Parse(r.URL.String())
		if err != nil {
			return nil, err
		}
		baseURL.Path = ""
		pageURL = baseURL.String()
	}
	page, err := browser.Context(r.Context()).Page(proto.TargetCreateTarget{URL: pageURL})
	if err != nil {
		return nil, err
	}
	defer page.Close()

	for k, v := range t.localStorage {
		_, err = page.Eval("(k, v) => localStorage[k] = v", k, v)
		if err != nil {
			return nil, err
		}
	}

	if r.Header.Get("Cookie") != "" {
		cookies := getHeaderCookies(r)
		err = page.SetCookies(cookies)
		if err != nil {
			return nil, err
		}
	}

	// Same headers as defined in mangoprovider/scraper/util.go.
	// Only set Referer for now, don't set "Host" as the request will fail.
	headers := []string{"Referer"}
	headersMap := getRequestHeaderMap(r, headers)
	_, err = page.SetExtraHeaders(headersMap)
	if err != nil {
		return nil, err
	}

	err = page.Navigate(r.URL.String())
	if err != nil {
		return nil, err
	}

	// Some scrapers don't work with page.WaitLoad directly,
	// better to use some time as per their config
	if t.loadWait > 0.0 {
		select {
		case <-r.Context().Done():
			return nil, r.Context().Err()
		case <-time.After(t.loadWait):
		}
	} else {
		err = page.WaitLoad()
		if err != nil {
			return nil, err
		}
	}

	if action := action(r.Context()); action != nil {
		if err := action(page); err != nil {
			return nil, err
		}
	}

	html, err := page.HTML()
	if err != nil {
		return nil, err
	}
	return &http.Response{
		Body:       io.NopCloser(strings.NewReader(html)),
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"text/html"}},
	}, nil
}

// logger sends rod's browser download messages to the provider's log.
type logger struct{ log *mango.Logger }

func (l logger) Println(vs ...any) {
	l.log.Log("%s", strings.TrimSpace(fmt.Sprintln(vs...)))
}
