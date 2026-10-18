package rod

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
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

// errClosed: The transport was closed, it doesn't launch the browser again.
var errClosed = errors.New("headless browser closed")

// downloadTimeout: How long the chromium download (~150 MB) can take.
var downloadTimeout = 10 * time.Minute

// responseWait: How long to wait for the page's response event once it's loaded.
const responseWait = 2 * time.Second

// Options: How the pages are loaded.
type Options struct {
	// LoadWait: How long to wait for the page to load, instead of its load event (e.g. html
	// filled in by scripts after it).
	LoadWait time.Duration
	// LocalStorage: Values set in the site's local storage before the request.
	LocalStorage map[string]string
}

// TransportRod is an http.RoundTripper that loads the pages in a headless browser, launched
// (and downloaded if needed) on the first request. The response is the page's html once
// loaded, with the status and headers of its document response.
type TransportRod struct {
	options Options
	log     *mango.Logger
	// newLauncher: launcher.New, replaced in tests
	newLauncher func() *launcher.Launcher

	// ctx: done once closed, it stops a launch in progress
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	browser  *rod.Browser
	launcher *launcher.Launcher
}

// NewTransport: The browser isn't launched until the first request.
func NewTransport(options Options, log *mango.Logger) *TransportRod {
	ctx, cancel := context.WithCancel(context.Background())
	return &TransportRod{
		options:     options,
		log:         log,
		newLauncher: launcher.New,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Close stops a launch in progress, closes the browser (killed if that fails) and removes its
// user data dir. Requests after it fail.
func (t *TransportRod) Close() error {
	t.cancel()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.browser == nil {
		return nil
	}
	err := t.browser.Close()
	if err != nil {
		t.launcher.Kill()
	}
	cleanup(t.launcher)
	t.browser, t.launcher = nil, nil
	return err
}

// launch: The browser, started on the first call. A failed launch is tried again by the next
// call; ctx (or Close) cancels the launch, not the launched browser; only Close (or
// downloadTimeout) cancels the download.
func (t *TransportRod) launch(ctx context.Context) (*rod.Browser, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ctx.Err() != nil {
		return nil, errClosed
	}
	if t.browser != nil {
		return t.browser, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(t.ctx, cancel)
	defer stop()

	// the launch and cleanup are duplicated in luaprovider's lib/headless (no shared module
	// but libmangal, which shouldn't pull rod): keep both in sync.
	// leakless (kills chromium if mangal dies) only on linux: antiviruses flag its binary on
	// windows
	l := t.newLauncher().Leakless(runtime.GOOS == "linux").Context(ctx)
	if l.Get(flags.Bin) == "" {
		// rod's own download logs to stdout, which must stay clean (e.g. --json)
		// the download outlives the request's deadline (the next request uses it), only Close
		// stops it
		downloadCtx, cancelDownload := context.WithTimeout(context.WithoutCancel(ctx), downloadTimeout)
		defer cancelDownload()
		stopDownload := context.AfterFunc(t.ctx, cancelDownload)
		defer stopDownload()
		b := launcher.NewBrowser()
		b.Context = downloadCtx
		b.Logger = logger{t.log}
		if b.Validate() != nil {
			t.log.Status(ctx, "downloading chromium for the headless browser")
		}
		bin, err := b.Get()
		if err != nil {
			return nil, t.launchErr(ctx, fmt.Errorf("download browser: %w", err))
		}
		l = l.Bin(bin)
	}
	t.log.Status(ctx, "starting the headless browser")
	u, err := l.Launch()
	if err != nil {
		// a browser that started but didn't answer is killed by Launch
		if l.PID() != 0 {
			cleanup(l)
		}
		return nil, t.launchErr(ctx, fmt.Errorf("launch browser: %w", err))
	}
	// the browser outlives the request, its connection doesn't take the request's ctx
	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		l.Kill()
		cleanup(l)
		return nil, fmt.Errorf("connect to browser: %w", err)
	}
	if ctx.Err() != nil {
		_ = browser.Close()
		cleanup(l)
		return nil, t.launchErr(ctx, ctx.Err())
	}
	t.browser, t.launcher = browser, l
	return browser, nil
}

// launchErr: err, or errClosed if Close stopped the launch.
func (t *TransportRod) launchErr(ctx context.Context, err error) error {
	if t.ctx.Err() != nil {
		return errClosed
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// RoundTrip loads the page in the browser, its html is the body.
func (t *TransportRod) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method != http.MethodGet {
		return nil, fmt.Errorf("headless browser: %s not supported, only GET", r.Method)
	}
	browser, err := t.launch(r.Context())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// only start on the site when there are local storage values to set
	startURL := ""
	if len(t.options.LocalStorage) != 0 {
		start := *r.URL
		start.Path, start.RawPath, start.RawQuery, start.Fragment = "", "", "", ""
		startURL = start.String()
	}
	page, err := browser.Context(ctx).Page(proto.TargetCreateTarget{URL: startURL})
	if err != nil {
		return nil, err
	}
	defer page.Close()
	if startURL != "" {
		if err := page.WaitLoad(); err != nil {
			return nil, err
		}
		for k, v := range t.options.LocalStorage {
			if _, err := page.Eval("(k, v) => localStorage[k] = v", k, v); err != nil {
				return nil, err
			}
		}
	}

	if cookies := requestCookies(r); len(cookies) != 0 {
		if err := page.SetCookies(cookies); err != nil {
			return nil, err
		}
	}

	// the first document response of the page from here on is the one navigated to (redirects
	// don't get one)
	responses := make(chan *proto.NetworkResponse, 1)
	go page.EachEvent(func(e *proto.NetworkResponseReceived) bool {
		if e.Type != proto.NetworkResourceTypeDocument || e.FrameID != page.FrameID {
			return false
		}
		responses <- e.Response
		return true
	})()

	nav, err := proto.PageNavigate{URL: r.URL.String(), Referrer: r.Referer()}.Call(page)
	if err != nil {
		return nil, err
	}
	// chromium refuses to show some error pages (e.g. a 429 without a body), the status is
	// still in the response
	failed := nav.ErrorText != ""
	if failed && nav.ErrorText != "net::ERR_HTTP_RESPONSE_CODE_FAILURE" {
		return nil, fmt.Errorf("navigate to %s: %s", r.URL, nav.ErrorText)
	}

	var html string
	if !failed {
		if err := t.waitLoad(ctx, page); err != nil {
			return nil, err
		}
		if action := action(r.Context()); action != nil {
			if err := action(page); err != nil {
				return nil, err
			}
		}
		if html, err = page.HTML(); err != nil {
			return nil, err
		}
	}

	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{},
		Body:          io.NopCloser(strings.NewReader(html)),
		ContentLength: int64(len(html)),
		Request:       r,
	}
	select {
	case doc := <-responses:
		resp.StatusCode = doc.Status
		for k, v := range doc.Headers {
			for value := range strings.SplitSeq(v.Str(), "\n") {
				resp.Header.Add(k, value)
			}
		}
		if u, err := url.Parse(doc.URL); err == nil && u.IsAbs() {
			// where the browser ended up, so redirects are seen like with a direct request
			resp.Request = r.Clone(r.Context())
			resp.Request.URL, resp.Request.Host = u, u.Host
		}
	case <-time.After(responseWait):
		if failed {
			return nil, fmt.Errorf("navigate to %s: %s", r.URL, nav.ErrorText)
		}
		t.log.Log("no response status for %s, assuming 200", r.URL)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	resp.Status = strconv.Itoa(resp.StatusCode) + " " + http.StatusText(resp.StatusCode)
	// the body is the page's html, always utf-8 and not compressed anymore
	for _, h := range []string{"Content-Length", "Content-Encoding", "Transfer-Encoding"} {
		resp.Header.Del(h)
	}
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")
	return resp, nil
}

// waitLoad: Waits LoadWait if set, else the page's load event.
func (t *TransportRod) waitLoad(ctx context.Context, page *rod.Page) error {
	if t.options.LoadWait <= 0 {
		return page.WaitLoad()
	}
	timer := time.NewTimer(t.options.LoadWait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// cleanup waits for the browser to exit (killed if it takes too long) and removes its user
// data dir.
func cleanup(l *launcher.Launcher) {
	exited := make(chan struct{})
	go func() {
		l.Cleanup()
		close(exited)
	}()
	select {
	case <-exited:
		return
	case <-time.After(10 * time.Second):
	}
	l.Kill()
	select {
	case <-exited:
	// rod's exit signal can be missing (e.g. leakless failed), the dir is all that's left
	case <-time.After(5 * time.Second):
		_ = os.RemoveAll(l.Get(flags.UserDataDir))
	}
}

// logger sends rod's browser download messages to the provider's log.
type logger struct{ log *mango.Logger }

func (l logger) Println(vs ...any) {
	l.log.Log("%s", strings.TrimSpace(fmt.Sprintln(vs...)))
}
