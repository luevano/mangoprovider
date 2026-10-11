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
		Version:     "0.3.0",
		Description: "Mangabat (needs FlareSolverr to search)",
		Website:     "https://www.mangabats.com/",
	}
	MangakakalotInfo = libmangal.ProviderInfo{
		ID:          mango.BundleID + "-mangakakalot",
		Name:        "Mangakakalot",
		Version:     "0.3.0",
		Description: "Mangakakalot (needs FlareSolverr to search)",
		Website:     "https://www.mangakakalove.com/",
	}
	ManganatoInfo = libmangal.ProviderInfo{
		ID:          mango.BundleID + "-manganato",
		Name:        "Manganato",
		Version:     "0.4.0",
		Description: "Manganato (needs FlareSolverr)",
		Website:     "https://www.natomanga.com/",
	}
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	return []libmangal.ProviderLoader{
		loader(MangabatInfo, options),
		loader(MangakakalotInfo, options),
		loader(ManganatoInfo, options),
	}
}

// flaresolverrTimeout: solving a challenge takes 10-20s, more when it's slow
const flaresolverrTimeout = 90 * time.Second

type mangabox struct {
	info      libmangal.ProviderInfo
	client    *http.Client
	userAgent string
	headless  mango.HeadlessOptions
	// 2 requests per second, like keiyoushi's
	limiter *rate.Limiter
	// base is the website, a test server in tests
	base string

	mu sync.Mutex
	// flaresolverr is created on the first challenge, its session closed with the provider
	flaresolverr *flaresolverr.TransportFlaresolverr
}

func loader(info libmangal.ProviderInfo, options mango.Options) libmangal.ProviderLoader {
	return &mango.Loader{
		ProviderInfo: info,
		Options:      options,
		F: func(context.Context, *mango.Logger) (mango.Functions, error) {
			m := &mangabox{
				info:      info,
				client:    options.HTTPClient,
				userAgent: options.UserAgent,
				headless:  options.Headless,
				limiter:   rate.NewLimiter(rate.Every(500*time.Millisecond), 2),
				base:      info.Website,
			}
			return mango.Functions{
				SearchMangas:   m.SearchMangas,
				MangaVolumes:   m.MangaVolumes,
				VolumeChapters: m.VolumeChapters,
				ChapterPages:   m.ChapterPages,
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
// challenges the request.
func (m *mangabox) get(ctx context.Context, path string) ([]byte, error) {
	if err := m.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	url := m.base + strings.TrimPrefix(path, "/")
	body, status, err := m.do(ctx, m.client, url)
	if err != nil {
		return nil, err
	}
	if challenged(status, body) {
		fs, err := m.flaresolverrClient()
		if err != nil {
			return nil, fmt.Errorf("GET /%s: %w", strings.TrimPrefix(path, "/"), err)
		}
		ctx, cancel := context.WithTimeout(ctx, flaresolverrTimeout)
		defer cancel()
		if body, status, err = m.do(ctx, fs, url); err != nil {
			return nil, fmt.Errorf("GET /%s: %w", strings.TrimPrefix(path, "/"), err)
		}
	}
	switch {
	case status == http.StatusNotFound:
		return nil, fmt.Errorf("GET /%s: %w", strings.TrimPrefix(path, "/"), mango.ErrNotFound)
	case status != http.StatusOK:
		return nil, fmt.Errorf("GET /%s: %d %s", strings.TrimPrefix(path, "/"), status, strings.ToLower(http.StatusText(status)))
	}
	return body, nil
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
