package rod

import (
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
	"github.com/go-rod/rod/lib/proto"
	mango "github.com/luevano/mangoprovider"
)

var _ http.RoundTripper = (*TransportRod)(nil)

// TransportRod implementation of Transport used for colly.
type TransportRod struct {
	// launched on the first request, so errors can be returned instead of panicking
	browser    *rod.Browser
	launchErr  error
	launchOnce sync.Once

	loadWait     time.Duration
	localStorage map[string]string
	actions      map[ActionType]Action
}

func (t *TransportRod) Close() error {
	if t.browser == nil {
		return nil
	}
	return t.browser.Close()
}

// NewTransport creates a new transport with the browser setup.
func NewTransport(
	loadWait time.Duration,
	localStorage map[string]string,
	actions map[ActionType]Action,
) *TransportRod {
	return &TransportRod{
		loadWait:     loadWait,
		localStorage: localStorage,
		actions:      actions,
	}
}

// launch: Starts the browser once, on the first request.
func (t *TransportRod) launch() (*rod.Browser, error) {
	t.launchOnce.Do(func() {
		u, err := launcher.New().Leakless(runtime.GOOS == "linux").Launch()
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

// RoundTrip gets called on each colly request.
func (t *TransportRod) RoundTrip(r *http.Request) (*http.Response, error) {
	browser, err := t.launch()
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

	// Once loaded, check if there are any actions that need to be execued.
	actionType := ActionType(r.Header.Get(ActionTypeHeader))
	action, ok := t.actions[actionType]
	if ok && action != nil {
		mango.Log("found action for %s", actionType)
		err := action(page)
		if err != nil {
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
