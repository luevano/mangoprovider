package rod

import (
	"net/http"
	"strings"

	"github.com/go-rod/rod/lib/proto"
)

// getRequestHeaderMap translates the http.request.Header into a "map" for rod.
// A "map"  here really just pairs of strings next to each other in a slice.
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

// getHeaderCookies translates header cookies "name1=value1,name2=value2" into the rod format.
// The URL needs to be set to the http.request.URL.
func getHeaderCookies(request *http.Request) []*proto.NetworkCookieParam {
	cookies := []*proto.NetworkCookieParam{}
	cookiesSplit := strings.Split(request.Header.Get("Cookie"), ",")

	for _, _cookie := range cookiesSplit {
		cookieSplit := strings.Split(_cookie, "=")
		// Only when the correct format is set
		if len(cookieSplit) == 2 {
			name := cookieSplit[0]
			value := cookieSplit[1]
			cookie := proto.NetworkCookieParam{
				Name:  name,
				Value: value,
				URL:   request.URL.String(),
			}
			cookies = append(cookies, &cookie)
		}
	}

	return cookies
}
