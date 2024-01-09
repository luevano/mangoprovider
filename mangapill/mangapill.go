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
	Version:     "0.1.0",
	Description: "Mangapill scraper",
	Website:     "https://mangapill.com/",
}

var scraperOptions = &scraper.Options{
	// Name:            "Mangapill",
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
		Name: func(selection *goquery.Selection) string {
			return strings.TrimSpace(selection.Find("div a div.leading-tight").Text())
		},
		URL: func(selection *goquery.Selection) string {
			return selection.Find("div a:first-child").AttrOr("href", "")
		},
		Cover: func(selection *goquery.Selection) string {
			return selection.Find("img").AttrOr("data-src", "")
		},
	},
	// TODO: update these for libmangal/mangoprovider and add VolumeExtractor
	ChapterExtractor: &scraper.ChapterExtractor{
		Selector: "div[data-filter-list] a",
		Name: func(selection *goquery.Selection) string {
			return strings.TrimSpace(selection.Text())
		},
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("href", "")
		},
		Volume: func(selection *goquery.Selection) string {
			return ""
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

	// TODO: use mangodex get chapter page for downloading, instead of the mangoloader generic one
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
