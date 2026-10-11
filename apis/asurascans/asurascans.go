// Package asurascans uses the JSON API behind Asura Scans' site (asurascans.com, was
// asuracomic.net, a WordPress theme before that).
package asurascans

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	Version:     "0.4.2",
	Description: "AsuraScans, from the site's API",
	Website:     website,
}

type asura struct {
	client    *http.Client
	userAgent string
	filter    mango.FilterOptions
	// limiter keeps to the 2 requests / 2 s the site tolerates (keiyoushi's limit), across loads
	limiter *rate.Limiter
	// base is the api, a test server in tests
	base string
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	a := asura{
		client:    options.HTTPClient,
		userAgent: options.UserAgent,
		filter:    options.Filter,
		limiter:   rate.NewLimiter(rate.Every(time.Second), 2),
		base:      api,
	}
	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func(context.Context, *mango.Logger) (mango.Functions, error) {
			return mango.Functions{
				SearchMangas:   a.SearchMangas,
				MangaByID:      a.MangaByID,
				MangaVolumes:   a.MangaVolumes,
				VolumeChapters: a.VolumeChapters,
				ChapterPages:   a.ChapterPages,
			}, nil
		},
	}
}

// get decodes the api response at path into v.
func (a *asura) get(ctx context.Context, path string, v any) error {
	if err := a.limiter.Wait(ctx); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", a.userAgent)
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("GET /%s: %w", path, mango.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET /%s: %d %s", path, resp.StatusCode, strings.ToLower(http.StatusText(resp.StatusCode)))
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("GET /%s: %w", path, err)
	}
	return nil
}
