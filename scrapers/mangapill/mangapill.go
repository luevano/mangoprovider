package mangapill

import (
	"fmt"
	"net/url"
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
	Version:     "0.4.0",
	Description: "Mangapill scraper",
	Website:     "https://mangapill.com/",
}

var Config = &scraper.Configuration{
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
		ID: func(_url string) string {
			return strings.Join(strings.Split(_url, "/")[4:], "/")
		},
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
		// id is constructed similar to manga id above, <number>/<chapter-name>
		ID: func(_url string) string {
			return strings.Join(strings.Split(_url, "/")[4:], "/")
		},
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("href", "")
		},
		Date: func(_ *goquery.Selection) libmangal.Date {
			// mangapill doesn't provide dates, just use scraping day
			// else it will just use anilist publication day
			today := time.Now()
			return libmangal.Date{
				Year:  today.Year(),
				Month: int(today.Month()),
				Day:   today.Day(),
			}
		},
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
