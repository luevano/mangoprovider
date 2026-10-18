// Package mangabox reads the Mangabox sites (Mangabat, Mangakakalot, Manganato), after their
// 2025 move to new domains (ported from keiyoushi's lib-multisrc/mangabox). Search is behind a
// Cloudflare challenge, so is everything on some sites: those requests go through FlareSolverr
// (rod's Chromium without it, which these challenges stop too).
package mangabox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/fetch"
	"golang.org/x/time/rate"
)

var (
	MangabatInfo = libmangal.ProviderInfo{
		ID:          mango.BundleID + "-mangabat",
		Name:        "Mangabat",
		Version:     "0.3.6",
		Description: "Mangabat (needs FlareSolverr to search)",
		Website:     "https://www.mangabats.com/",
	}
	// .gg's pages are challenged and only a FlareSolverr on Google Chrome solves them
	MangakakalotInfo = libmangal.ProviderInfo{
		ID:          mango.BundleID + "-mangakakalot",
		Name:        "Mangakakalot",
		Version:     "0.3.6",
		Description: "Mangakakalot (needs FlareSolverr to search)",
		Website:     "https://www.mangakakalove.com/",
		Mirrors:     []string{"https://www.mangakakalot.gg/"},
	}
	ManganatoInfo = libmangal.ProviderInfo{
		ID:          mango.BundleID + "-manganato",
		Name:        "Manganato",
		Version:     "0.4.6",
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

type mangabox struct {
	info libmangal.ProviderInfo
	// client: direct, retried through FlareSolverr (or rod) on a challenge, 2 requests per
	// second: keiyoushi's Mangakakalot.kt rateLimit(2, 1.seconds) (src/en/mangakakalot), the
	// lib-multisrc/mangabox base and the other two set none
	client *fetch.Client
	// images: Optional, page images wait for it (see imageLimiter)
	images    *rate.Limiter
	userAgent string
	mirrors   *mango.Mirrors
	log       *mango.Logger

	mu sync.Mutex
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
			client, err := fetch.NewClient(options.HTTPClient, options.Headless, log, fetch.Config{
				Strategy: fetch.HeadlessFallback,
				Limiter:  rate.NewLimiter(rate.Every(500*time.Millisecond), 2),
				Retry:    &fetch.Retry{},
				Debug:    options.Debug,
			})
			if err != nil {
				return mango.Functions{}, err
			}
			m := &mangabox{
				info:      info,
				client:    client,
				images:    imageLimiter(info),
				userAgent: options.UserAgent,
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
				Close:          m.client.Close,
			}, nil
		},
	}
}

// imageLimiter: keiyoushi's Mangakakalot limit covers every request, images included; the
// other sites' images aren't limited.
func imageLimiter(info libmangal.ProviderInfo) *rate.Limiter {
	if info.ID != MangakakalotInfo.ID {
		return nil
	}
	return rate.NewLimiter(rate.Every(500*time.Millisecond), 2)
}

// get the page at path (relative to the website), in a headless browser when Cloudflare
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
	body, resp, err := mango.Fetch(ctx, m.client.Client, nil, m.userAgent, url)
	if err != nil {
		return nil, down(ctx, err)
	}
	if err := mango.StatusError(resp.StatusCode, resp.Header); err != nil {
		return nil, err
	}
	return body, nil
}

// down marks a request error as the site being down, unless it was cancelled.
func down(ctx context.Context, err error) error {
	if ctx.Err() != nil || errors.Is(err, mango.ErrSiteDown) {
		return err
	}
	return fmt.Errorf("%w: %w", mango.ErrSiteDown, err)
}
