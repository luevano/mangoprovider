package headless

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/flaresolverr"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

// Transport defines a transport used for colly.
type Transport interface {
	http.RoundTripper
	io.Closer
}

// NewTransport returns a flaresolverr transport if it's configured and reachable, else a rod one.
func NewTransport(
	ctx context.Context,
	options mango.HeadlessOptions,
	loadWait time.Duration,
	localStorage map[string]string,
	actions map[rod.ActionType]rod.Action,
	log *mango.Logger,
) Transport {
	if options.UseFlaresolverr && options.FlaresolverrURL != "" {
		if flaresolverrReachable(ctx, options.FlaresolverrURL) {
			return flaresolverr.NewTransport(options.FlaresolverrURL)
		}
		log.Status(ctx, "couldn't connect to flaresolverr, falling back to rod")
	}
	return rod.NewTransport(loadWait, localStorage, actions, log)
}

func flaresolverrReachable(ctx context.Context, flaresolverrURL string) bool {
	u, err := url.Parse(flaresolverrURL)
	if err != nil {
		return false
	}
	u.Path = ""

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
