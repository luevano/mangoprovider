package mangapill

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangapill",
	Name:        "Mangapill",
	Version:     "0.4.2",
	Description: "Mangapill scraper",
	Website:     "https://mangapill.com/",
}

// Config: A new configuration, built on every load.
func Config(_ *mango.Logger) *scraper.Configuration {
	return &scraper.Configuration{
		Name:            Info.ID,
		Delay:           50 * time.Millisecond,
		ReverseChapters: true,
		BaseURL:         Info.Website,
		GenerateSearchURL: func(baseUrl string, query string) (string, error) {
			// path is /search?q=<query>&type=&status=
			params := url.Values{}
			params.Set("q", query)
			params.Set("type", "")
			params.Set("status", "")

			u, _ := url.Parse(baseUrl)
			u.Path = "/search"
			u.RawQuery = params.Encode()

			return u.String(), nil
		},
		// keiyoushi's: the "Next" button, ?page=N
		SearchPagination: &scraper.Pagination{
			NextSelector: "a.btn.btn-sm",
			URL: func(searchURL string, page int) (string, error) {
				u, err := url.Parse(searchURL)
				if err != nil {
					return "", err
				}
				params := u.Query()
				params.Set("page", strconv.Itoa(page))
				u.RawQuery = params.Encode()
				return u.String(), nil
			},
		},
		GenerateSearchByIDURL: func(baseUrl string, id string) (string, error) {
			return fmt.Sprintf("%smanga/%s", baseUrl, id), nil
		},
		MangaByIDExtractor: &scraper.MangaByIDExtractor{
			Selector: "div.container > div.flex.flex-col.sm\\:flex-row.my-3",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("div.flex.flex-col > div.mb-3 > h1").Text()
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("div.text-transparent > img").AttrOr("data-src", "")
			},
		},
		MangaExtractor: &scraper.MangaExtractor{
			Selector: "body > div.container.py-3 > div.my-3.grid.justify-end.gap-3.grid-cols-2.md\\:grid-cols-3.lg\\:grid-cols-5 > div",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("div a div.leading-tight").Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find("div a:first-child").AttrOr("href", "")
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("img").AttrOr("data-src", "")
			},
			// the id is in the form <number>/<manga-name> as with just <number>
			// the url returns 404 (in case it's needed)
			ID: pathID,
		},
		VolumeExtractor: &scraper.VolumeExtractor{
			// selector that points to only 1 element ("Chapters" header)
			Selector: "body > div.container > div.border.border-border.rounded > div.p-3.border-b.border-border > div.flex.flex-col.md\\:flex-row.md\\:items-center.md\\:justify-between",
			Number: func(selection *goquery.Selection) float32 {
				return 1.0
			},
		},
		ChapterExtractor: &scraper.ChapterExtractor{
			Selector: "div[data-filter-list] a",
			Title: func(selection *goquery.Selection) string {
				return selection.Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("href", "")
			},
			// id is constructed similar to manga id above, <number>/<chapter-name>
			ID: pathID,
			ScanlationGroup: func(_ *goquery.Selection) string {
				return Info.Name
			},
		},
		PageExtractor: &scraper.PageExtractor{
			Selector: "picture img",
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("data-src", "")
			},
		},
	}
}

// pathID: The path after its first part, e.g. "<number>/<name>" for /manga/<number>/<name>.
func pathID(_url string) string {
	if parts := scraper.PathParts(_url); len(parts) >= 2 {
		return strings.Join(parts[1:], "/")
	}
	return ""
}
