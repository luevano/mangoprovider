package rod

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	mango "github.com/luevano/mangoprovider"
)

var _ http.RoundTripper = (*TransportRod)(nil)

// TransportRod implementation of Transport used for colly.
type TransportRod struct {
	browser      *rod.Browser
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
func NewTransport(localStorage map[string]string, actions map[ActionType]Action) *TransportRod {
	u := launcher.New().Leakless(runtime.GOOS == "linux").MustLaunch()
	return &TransportRod{
		browser:      rod.New().ControlURL(u).MustConnect(),
		localStorage: localStorage,
		actions:      actions,
	}
}

// RoundTrip gets called on each colly request.
func (t *TransportRod) RoundTrip(r *http.Request) (*http.Response, error) {
	var page *rod.Page
	// Only use the base URL when there are local storage values to set
	if len(t.localStorage) == 0 {
		page = t.browser.Context(r.Context()).MustPage("")
	} else {
		baseURL, err := url.Parse(r.URL.String())
		if err != nil {
			return nil, err
		}
		baseURL.Path = ""
		page = t.browser.Context(r.Context()).MustPage(baseURL.String())

		for k, v := range t.localStorage {
			_, err = page.Eval("(k, v) => localStorage[k] = v", k, v)
			if err != nil {
				return nil, err
			}
		}
	}
	defer page.Close()

	if r.Header.Get("Cookie") != "" {
		cookies := getHeaderCookies(r)
		page = page.MustSetCookies(cookies...)
	}

	// Same headers as defined in mangoprovider/scraper/util.go.
	// Only set Referer for now, don't set "Host" as the request will fail.
	headers := []string{"Referer"}
	headersMap := getRequestHeaderMap(r, headers)
	_ = page.MustSetExtraHeaders(headersMap...)

	page = page.MustNavigate(r.URL.String()).MustWaitLoad()
	// More than enough time to let the page actually load
	time.Sleep(1 * time.Second)

	// Once loaded, check if there are any actions that need to be execued.
	actionType := ActionType(r.Header.Get(ActionTypeHeader))
	action, ok := t.actions[actionType]
	if ok && action != nil {
		mango.Log(fmt.Sprintf("found action for %s", actionType))
		err := action(page)
		if err != nil {
			return nil, err
		}
	}

	return &http.Response{
		Body:       newPageReader(page),
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"text/html"}},
	}, nil
}
