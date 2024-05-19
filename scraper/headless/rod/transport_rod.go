package rod

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
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
	var err error
	// Only use the base URL when there are local storage values to set
	if len(t.localStorage) == 0 {
		page, err = t.browser.Context(r.Context()).Page(proto.TargetCreateTarget{URL: ""})
		if err != nil {
			return nil, err
		}
	} else {
		baseURL, err := url.Parse(r.URL.String())
		if err != nil {
			return nil, err
		}
		baseURL.Path = ""
		page, err = t.browser.Context(r.Context()).Page(proto.TargetCreateTarget{URL: baseURL.String()})
		if err != nil {
			return nil, err
		}

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

	// TODO: make this configurable
	//
	// Looks like it is necessary to wait for some time (at least for asurascans),
	// no other kind of page.WaitX seems to work, just raw time.Sleep.
	// Has to be done before any kind of page.WaitX. Still using WaitLoad in case it's needed.
	time.Sleep(2 * time.Second)
	err = page.WaitLoad()
	if err != nil {
		return nil, err
	}

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
