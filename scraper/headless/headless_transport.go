package headless

import (
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/luevano/mangoprovider/scraper/headless/flaresolverr"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

var (
	transportInstance TransportHeadless
	once              sync.Once
)

type TransportHeadless interface {
	http.RoundTripper
	io.Closer
}

// IsLoaded returns true if the headless transport is loaded
func IsLoaded() bool {
	return transportInstance != nil
}

func GetTransportSingleton(options Options) TransportHeadless {
	once.Do(func() {
		if options.UseFlaresolverr && options.FlaresolverrURL != "" {
			url, err := url.Parse(options.FlaresolverrURL)
			if err != nil {
				// log.Error("Couldn't parse flaresolverr url, falling back to rod")
				transportInstance = rod.NewTransport()
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
				// log.Error("Couldn't connect to flaresolverr, falling back to rod")
				transportInstance = rod.NewTransport()
				return
			}
			transportInstance = flaresolverr.NewTransport(options.FlaresolverrURL)
			return
		}
		transportInstance = rod.NewTransport()
	})
	return transportInstance
}
