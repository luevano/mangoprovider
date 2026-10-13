// Package mangabox reads the Mangabox sites (Mangabat, Mangakakalot, Manganato), after their
// 2025 move to new domains (ported from keiyoushi's lib-multisrc/mangabox). Search is behind a
// Cloudflare challenge, so is everything on some sites: those requests go through FlareSolverr.
package mangabox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/flaresolverr"
	"golang.org/x/time/rate"
)

var (
	MangabatInfo = libmangal.ProviderInfo{
		ID:          mango.BundleID + "-mangabat",
		Name:        "Mangabat",
		Version:     "0.3.2",
		Description: "Mangabat (needs FlareSolverr to search)",
		Website:     "https://www.mangabats.com/",
	}
	// .gg's pages are challenged and only a FlareSolverr on Google Chrome solves them
	MangakakalotInfo = libmangal.ProviderInfo{
		ID:          mango.BundleID + "-mangakakalot",
		Name:        "Mangakakalot",
		Version:     "0.3.2",
		Description: "Mangakakalot (needs FlareSolverr to search)",
		Website:     "https://www.mangakakalove.com/",
		Mirrors:     []string{"https://www.mangakakalot.gg/"},
	}
	ManganatoInfo = libmangal.ProviderInfo{
		ID:          mango.BundleID + "-manganato",
		Name:        "Manganato",
		Version:     "0.4.2",
		Description: "Manganato (needs FlareSolverr)",
		Website:     "https://www.natomanga.com/",
		Mirrors:     []string{"https://www.nelomanga.net/", "https://www.manganato.gg/"},
	}
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	return []libmangal.ProviderLoader{
		loader(MangabatInfo, options),
		loader(MangakakalotInfo, options),
		loader(ManganatoInfo, options),
	}
}

// flaresolverrTimeout: solving a challenge takes 10-20s; with mirrors, an unsolvable one
// shouldn't hold the next for long
const flaresolverrTimeout = 60 * time.Second

type mangabox struct {
	info      libmangal.ProviderInfo
	client    *http.Client
	userAgent string
	headless  mango.HeadlessOptions
	// 2 requests per second, like keiyoushi's
	limiter *rate.Limiter
	mirrors *mango.Mirrors
	log     *mango.Logger

	mu sync.Mutex
	// flaresolverr is created on the first challenge, its session closed with the provider
	flaresolverr *flaresolverr.TransportFlaresolverr
	// cdn: the origin of the cdn that served the last image, tried first
	cdn string
	// cdnDown: cdns already reported failing, until one of their images works again
	cdnDown map[string]bool
}

func loader(info libmangal.ProviderInfo, options mango.Options) libmangal.ProviderLoader {
	return &mango.Loader{
		ProviderInfo: info,
		Options:      options,
		F: func(_ context.Context, log *mango.Logger) (mango.Functions, error) {
			m := &mangabox{
				info:      info,
				client:    options.HTTPClient,
				userAgent: options.UserAgent,
				headless:  options.Headless,
				limiter:   rate.NewLimiter(rate.Every(500*time.Millisecond), 2),
				mirrors:   mango.NewMirrors(info, options.Mirrors[info.ID], log),
				log:       log,
				cdnDown:   map[string]bool{},
			}
			return mango.Functions{
				SearchMangas:   m.SearchMangas,
				MangaByID:      m.MangaByID,
				MangaVolumes:   m.MangaVolumes,
				VolumeChapters: m.VolumeChapters,
				ChapterPages:   m.ChapterPages,
				GetPageImage:   m.GetPageImage,
				Close:          m.Close,
			}, nil
		},
	}
}

func (m *mangabox) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flaresolverr == nil {
		return nil
	}
	return m.flaresolverr.Close()
}

// get the page at path (relative to the website), through FlareSolverr when Cloudflare
// challenges the request, from the first mirror that's up. Returns the website it came from.
func (m *mangabox) get(ctx context.Context, store mango.Store, path string) (body []byte, website string, err error) {
	path = strings.TrimPrefix(path, "/")
	err = m.mirrors.Do(ctx, store, func(site string) error {
		body, err = m.getFrom(ctx, site+path)
		if err != nil {
			return fmt.Errorf("GET /%s: %w", path, err)
		}
		website = site
		return nil
	})
	return body, website, err
}

func (m *mangabox) getFrom(ctx context.Context, url string) ([]byte, error) {
	if err := m.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	body, status, err := m.do(ctx, m.client, url)
	if err != nil {
		return nil, down(ctx, err)
	}
	if challenged(status, body) {
		fs, err := m.flaresolverrClient()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", mango.ErrSiteDown, err)
		}
		m.log.Status(ctx, "cloudflare challenge on %s, solving with flaresolverr (up to %ds)", mango.Host(url), int(flaresolverrTimeout.Seconds()))
		ctx, cancel := context.WithTimeout(ctx, flaresolverrTimeout)
		defer cancel()
		if body, status, err = m.do(ctx, fs, url); err != nil {
			return nil, down(ctx, err)
		}
	}
	switch {
	case status == http.StatusNotFound:
		return nil, mango.ErrNotFound
	case status >= 500:
		return nil, fmt.Errorf("%w: %d %s", mango.ErrSiteDown, status, strings.ToLower(http.StatusText(status)))
	case status != http.StatusOK:
		return nil, fmt.Errorf("%d %s", status, strings.ToLower(http.StatusText(status)))
	}
	return body, nil
}

// down marks a request error as the site being down, unless it was cancelled.
func down(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return err
	}
	return fmt.Errorf("%w: %w", mango.ErrSiteDown, err)
}

func (m *mangabox) do(ctx context.Context, client *http.Client, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", m.userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func (m *mangabox) flaresolverrClient() (*http.Client, error) {
	if !m.headless.UseFlaresolverr || m.headless.FlaresolverrURL == "" {
		return nil, fmt.Errorf("cloudflare challenge, enable flaresolverr (providers.headless.use_flaresolverr)")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flaresolverr == nil {
		m.flaresolverr = flaresolverr.NewTransport(m.headless.FlaresolverrURL)
	}
	return &http.Client{Transport: m.flaresolverr}, nil
}

func challenged(status int, body []byte) bool {
	return (status == http.StatusForbidden || status == http.StatusServiceUnavailable) &&
		bytes.Contains(body, []byte("<title>Just a moment...</title>"))
}
