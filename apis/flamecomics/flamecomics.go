// Package flamecomics reads Flame Comics' pages, a Next.js site: every page embeds its
// data as JSON (__NEXT_DATA__), so there's nothing to scrape from the html.
package flamecomics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
)

const (
	website = "https://flamecomics.xyz/"
	cdn     = "https://cdn.flamecomics.xyz/uploads/images/series/"
)

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-flamecomics",
	Name:        "FlameComics",
	Version:     "0.4.0",
	Description: "FlameComics, from the data its pages embed",
	Website:     website,
}

var nextData = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

type flame struct {
	client    *http.Client
	userAgent string
	// base is the website, a test server in tests
	base string
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	f := flame{client: options.HTTPClient, userAgent: options.UserAgent, base: website}
	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func(context.Context, *mango.Logger) (mango.Functions, error) {
			return mango.Functions{
				SearchMangas:   f.SearchMangas,
				MangaVolumes:   f.MangaVolumes,
				VolumeChapters: f.VolumeChapters,
				ChapterPages:   f.ChapterPages,
			}, nil
		},
	}
}

// page gets a page of the site and decodes the props it embeds into v.
func (f *flame) page(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", f.userAgent)
	resp, err := f.client.Do(req)
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
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
