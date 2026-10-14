// Package asurascans uses the JSON API behind Asura Scans' site (asurascans.com, was
// asuracomic.net, a WordPress theme before that).
package asurascans

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"golang.org/x/time/rate"
)

const (
	website = "https://asurascans.com/"
	api     = "https://api.asurascans.com/api/"
	// searchPages caps the search, 20 results each
	searchPages = 5
)

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-asurascans",
	Name:        "AsuraScans",
	Version:     "0.4.4",
	Description: "AsuraScans, from the site's API",
	Website:     website,
}

type asura struct {
	client    *http.Client
	userAgent string
	filter    mango.FilterOptions
	// limiter keeps to the 2 requests / 2 s the site tolerates, across loads: keiyoushi's
	// rateLimit(2, 2.seconds) on everything but the covers (src/en/asurascans/.../AsuraScans.kt),
	// the cdn's images included
	limiter *rate.Limiter
	// base is the api, a test server in tests
	base string
	log  *mango.Logger
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	limiter := rate.NewLimiter(rate.Every(time.Second), 2)
	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func(_ context.Context, log *mango.Logger) (mango.Functions, error) {
			a := &asura{
				client:    options.HTTPClient,
				userAgent: options.UserAgent,
				filter:    options.Filter,
				limiter:   limiter,
				base:      api,
				log:       log,
			}
			return mango.Functions{
				SearchMangas:   a.SearchMangas,
				MangaByID:      a.MangaByID,
				MangaVolumes:   a.MangaVolumes,
				VolumeChapters: a.VolumeChapters,
				ChapterPages:   a.ChapterPages,
				GetPageImage:   a.GetPageImage,
			}, nil
		},
	}
}

// GetPageImage is the generic one, throttled: the cdn answers 429 to unthrottled pages.
func (a *asura) GetPageImage(ctx context.Context, client *http.Client, page mango.Page) ([]byte, error) {
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return mango.GenericGetPageImage(ctx, client, page)
}

// get decodes the api response at path into v.
func (a *asura) get(ctx context.Context, path string, v any) error {
	body, resp, err := mango.Fetch(ctx, a.client, a.limiter, a.userAgent, a.base+path)
	if err != nil {
		return err
	}
	if err := mango.StatusError(resp.StatusCode, resp.Header); err != nil {
		return fmt.Errorf("GET /%s: %w", path, err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("GET /%s: %w", path, err)
	}
	return nil
}
