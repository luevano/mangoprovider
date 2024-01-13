package mangapill

import (
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/luevano/mangoprovider/mango/scraper"
)

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangapill",
	Name:        "Mangapill",
	Version:     "0.2.0",
	Description: "Mangapill scraper",
	Website:     "https://mangapill.com/",
}

var scraperOptions = &scraper.Options{
	Delay:           50 * time.Millisecond,
	Parallelism:     50,
	ReverseChapters: true,
	BaseURL:         providerInfo.Website,
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
			return strings.TrimSpace(selection.Find("div a div.leading-tight").Text())
		},
		URL: func(selection *goquery.Selection) string {
			return selection.Find("div a:first-child").AttrOr("href", "")
		},
		Cover: func(selection *goquery.Selection) string {
			return selection.Find("img").AttrOr("data-src", "")
		},
		ID: func(_url string) string {
			urlSplit := strings.Split(_url, "/")
			// TODO: should the ID be 123/manga-name instead of just 123?
			return urlSplit[4]
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
			return strings.TrimSpace(selection.Text())
		},
		ID: func(_url string) string {
			urlSplit := strings.Split(_url, "/")
			// TODO: should the ID be 123/chapter-name instead of just 123?
			return urlSplit[4]
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
		ScanlationGroup: func(selection *goquery.Selection) string {
			// mangapill doens't provide scanlators, just use "mangapill"
			// to avoid using anilist translators
			return "mangapill"
		},
	},
	PageExtractor: &scraper.PageExtractor{
		Selector: "picture img",
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("data-src", "")
		},
	},
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	s, err := scraper.NewScraper(scraperOptions)
	// TODO: panic?
	if err != nil {
		return nil
	}

	// TODO: use mangodex get chapter page for downloading,
	// instead of the mangoloader generic one
	return mango.ProviderLoader{
		ProviderInfo: providerInfo,
		Options:      options,
		Funcs: mango.ProviderFuncs{
			SearchMangas:   s.SearchMangas,
			MangaVolumes:   s.MangaVolumes,
			VolumeChapters: s.VolumeChapters,
			ChapterPages:   s.ChapterPages,
		},
	}
}
