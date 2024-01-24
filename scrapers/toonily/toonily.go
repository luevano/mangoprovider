package toonily

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
	ID:          mango.BundleID + "-toonily",
	Name:        "Toonily",
	Version:     "0.1.0",
	Description: "Toonily scraper",
	Website:     "https://toonily.com/",
}

var Config = &scraper.Configuration{
	Name:            Info.ID,
	Delay:           50 * time.Millisecond,
	ReverseChapters: true,
	Cookies:         "toonily-mature=1",
	BaseURL:         Info.Website,
	GenerateSearchURL: func(baseUrl string, query string) (string, error) {
		// path is /search/
		u, _ := url.Parse(baseUrl)
		query = strings.ReplaceAll(query, " ", "-")
		u.Path = fmt.Sprintf("/search/%s", query)

		return u.String(), nil
	},
	MangaExtractor: &scraper.MangaExtractor{
		Selector: ".page-item-detail.manga",
		Title: func(selection *goquery.Selection) string {
			return selection.Find(".item-summary .post-title a").Text()
		},
		URL: func(selection *goquery.Selection) string {
			return selection.Find(".item-summary .post-title a").AttrOr("href", "")
		},
		Cover: func(selection *goquery.Selection) string {
			return selection.Find("img").AttrOr("data-src", "")
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[4]
		},
	},
	VolumeExtractor: &scraper.VolumeExtractor{
		// selector that points to only 1 element
		Selector: "body",
		Number: func(selection *goquery.Selection) int {
			return 1
		},
	},
	ChapterExtractor: &scraper.ChapterExtractor{
		Selector: "div.listing-chapters_wrap > ul li",
		Title: func(selection *goquery.Selection) string {
			name := selection.Find("a").Text()
			return name
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[5]
		},
		URL: func(selection *goquery.Selection) string {
			return selection.Find("a").AttrOr("href", "")
		},
		Date: func(selection *goquery.Selection) libmangal.Date {
			layout := "Jan 02, 06"
			publishedDate := selection.Find("span.chapter-release-date i").Text()
			date, err := time.Parse(layout, publishedDate)
			if err != nil {
				// if failed to parse date, use scraping day
				date = time.Now()
			}
			return libmangal.Date{
				Year:  date.Year(),
				Month: int(date.Month()),
				Day:   date.Day(),
			}
		},
		ScanlationGroup: func(_ *goquery.Selection) string {
			return Info.Name
		},
	},
	PageExtractor: &scraper.PageExtractor{
		Selector: "div.reading-content div.page-break.no-gaps img",
		URL: func(selection *goquery.Selection) string {
			return strings.TrimSpace(selection.AttrOr("data-src", ""))
		},
	},
}
