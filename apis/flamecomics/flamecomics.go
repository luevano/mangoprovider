// Package flamecomics reads Flame Comics' pages, a Next.js site: every page embeds its
// data as JSON (__NEXT_DATA__), so there's nothing to scrape from the html.
package flamecomics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"golang.org/x/time/rate"
)

const (
	website = "https://flamecomics.xyz/"
	cdn     = "https://cdn.flamecomics.xyz/uploads/images/series/"
)

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-flamecomics",
	Name:        "FlameComics",
	Version:     "0.4.3",
	Description: "FlameComics, from the data its pages embed",
	Website:     website,
}

var nextData = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

type flame struct {
	client    *http.Client
	userAgent string
	// limiter: pages and page images, across loads (see newLimiter)
	limiter *rate.Limiter
	// base is the website, a test server in tests
	base string
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	f := flame{
		client:    options.HTTPClient,
		userAgent: options.UserAgent,
		limiter:   newLimiter(),
		base:      website,
	}
	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func(context.Context, *mango.Logger) (mango.Functions, error) {
			return mango.Functions{
				SearchMangas:   f.SearchMangas,
				MangaByID:      f.MangaByID,
				MangaVolumes:   f.MangaVolumes,
				VolumeChapters: f.VolumeChapters,
				ChapterPages:   f.ChapterPages,
				GetPageImage:   f.GetPageImage,
			}, nil
		},
	}
}

// newLimiter: 2 requests / 2 s, keiyoushi's rateLimit(2, 2.seconds) on everything but the
// covers (src/en/flamecomics/.../FlameComics.kt), the cdn's page images included.
func newLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(time.Second), 2)
}

// GetPageImage is the generic one, throttled.
func (f *flame) GetPageImage(ctx context.Context, client *http.Client, page mango.Page) ([]byte, error) {
	if err := f.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return mango.GenericGetPageImage(ctx, client, page)
}

// page gets a page of the site and decodes the props it embeds into v.
func (f *flame) page(ctx context.Context, path string, v any) error {
	body, resp, err := mango.Fetch(ctx, f.client, f.limiter, f.userAgent, f.base+path)
	if err != nil {
		return err
	}
	if err := mango.StatusError(resp.StatusCode, resp.Header); err != nil {
		return fmt.Errorf("GET /%s: %w", path, err)
	}
	m := nextData.FindSubmatch(body)
	if m == nil {
		return fmt.Errorf("GET /%s: no page data", path)
	}
	var data struct {
		Props struct {
			PageProps json.RawMessage `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(m[1], &data); err != nil {
		return fmt.Errorf("GET /%s: %w", path, err)
	}
	if err := json.Unmarshal(data.Props.PageProps, v); err != nil {
		return fmt.Errorf("GET /%s: %w", path, err)
	}
	return nil
}
