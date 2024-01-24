package mangapill

import (
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
	Version:     "0.3.1",
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
		Number: func(selection *goquery.Selection) int {
			return 1
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
