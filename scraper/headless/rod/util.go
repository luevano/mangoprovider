package rod

import (
	"net/http"

	"github.com/go-rod/rod/lib/proto"
)

// requestCookies: The request's cookies (its Cookie header, e.g. from the client's jar) for
// the browser.
func requestCookies(r *http.Request) []*proto.NetworkCookieParam {
	var cookies []*proto.NetworkCookieParam
	for _, c := range r.Cookies() {
		cookies = append(cookies, &proto.NetworkCookieParam{Name: c.Name, Value: c.Value, URL: r.URL.String()})
	}
	return cookies
}
