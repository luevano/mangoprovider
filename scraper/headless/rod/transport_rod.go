package rod

import (
	"net/http"
	"runtime"
	"strings"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var _ http.RoundTripper = (*TransportRod)(nil)

type TransportRod struct {
	browser        *rod.Browser
	browserBuilder sync.Once
}

func (t *TransportRod) Close() error {
	if t.browser == nil {
		return nil
	}
	return t.browser.Close()
}

func NewTransport() *TransportRod {
	return &TransportRod{}
}

// get the list of headers from the http request as a "map" (really just pairs of strings next to each other in a slice)
func getRequestHeaderMap(request *http.Request, headers []string) []string {
	var headerMap []string
	for _, header := range headers {
		if request.Header.Get(header) != "" {
			h := []string{header, request.Header.Get(header)}
			headerMap = append(headerMap, h...)
		}
	}
	return headerMap
}

// get cookies from header (name1=value1,name2=value2) into the rod format, the URL needs to be set
func getHeaderCookies(_cookies, url string) []*proto.NetworkCookieParam {
	cookies := []*proto.NetworkCookieParam{}
	cookiesSplit := strings.Split(_cookies, ",")

	for _, _cookie := range cookiesSplit {
		cookieSplit := strings.Split(_cookie, "=")
		// Only when the correct format is set
		if len(cookieSplit) == 2 {
			name := cookieSplit[0]
			value := cookieSplit[1]
			cookie := proto.NetworkCookieParam{
				Name:  name,
				Value: value,
				URL:   url,
			}
			cookies = append(cookies, &cookie)
		}
	}

	return cookies
}

func (t *TransportRod) RoundTrip(request *http.Request) (*http.Response, error) {
	// Only create a new browser once
	// TODO: move page creation in here and set required values as needed? Also set cookies on browser level?
	t.browserBuilder.Do(func() {
		u := launcher.New().Leakless(runtime.GOOS == "linux").MustLaunch()
		t.browser = rod.New().ControlURL(u).MustConnect()
	})
	page, err := t.browser.Context(request.Context()).Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, err
	}
	defer page.Close()

	if request.Header.Get("Cookie") != "" {
		cookies := getHeaderCookies(request.Header.Get("Cookie"), request.URL.String())
		err = page.SetCookies(cookies)
		if err != nil {
			return nil, err
		}
	}

	// Same headers as defined in mangoprovider/scraper/util.go, only set Referer for now
	headers := []string{"Referer"}
	headersMap := getRequestHeaderMap(request, headers)
	_, err = page.SetExtraHeaders(headersMap)
	if err != nil {
		return nil, err
	}

	err = page.Navigate(request.URL.String())
	if err != nil {
		return nil, err
	}

	err = page.WaitLoad()
	if err != nil {
		return nil, err
	}

	return &http.Response{
		Body:       newPageReader(page),
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"text/html"}},
	}, nil
}
