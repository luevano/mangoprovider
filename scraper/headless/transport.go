package headless

import (
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/flaresolverr"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

var (
	transport Transport
	once      sync.Once
)

// Transport defines a transport used for colly.
type Transport interface {
	http.RoundTripper
	io.Closer
}

// IsLoaded returns true if the transport is loaded.
func IsLoaded() bool {
	return transport != nil
}

// GetTransport returns the singleton rod or flaresolverr transport.
func GetTransport(
	options mango.HeadlessOptions,
	loadWait time.Duration,
	localStorage map[string]string,
	actions map[rod.ActionType]rod.Action,
) Transport {
	once.Do(func() {
		if options.UseFlaresolverr && options.FlaresolverrURL != "" {
			url, err := url.Parse(options.FlaresolverrURL)
			if err != nil {
				mango.Log("couldn't parse flaresolverr url, falling back to rod")
				transport = rod.NewTransport(loadWait, localStorage, actions)
				return
			}

			url.Path = ""
			result, err := http.Get(url.String())
			defer func() {
				if result != nil && result.Body != nil {
					result.Body.Close()
				}
			}()
			if err != nil || result.StatusCode != 200 {
				mango.Log("couldn't connect to flaresolverr, falling back to rod")
				transport = rod.NewTransport(loadWait, localStorage, actions)
				return
			}
			transport = flaresolverr.NewTransport(options.FlaresolverrURL)
			return
		}
		transport = rod.NewTransport(loadWait, localStorage, actions)
	})
	return transport
}
