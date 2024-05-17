package mangasee

import (
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangasee",
	Name:        "MangaSee",
	Version:     "0.1.0",
	Description: "MangaSee scraper",
	Website:     "https://mangasee123.com/",
}

var Config = &scraper.Configuration{
	Name:                 Info.ID,
	Delay:                50 * time.Millisecond,
	ReverseChapters:      true,
	NeedsHeadlessBrowser: true,
	BaseURL:              Info.Website,
	Headers: map[string]string{
		"Cookie": "FullPage=yes",
	},
	GenerateSearchURL: func(baseUrl string, query string) (string, error) {
		// path is /search/?name=
		params := url.Values{}
		params.Set("name", query)
		u, _ := url.Parse(baseUrl)
		u.Path = "/search/"
		u.RawQuery = params.Encode()

		return u.String(), nil
	},
	MangaExtractor: &scraper.MangaExtractor{
		Selector: ".top-15.ng-scope > .row",
		Title: func(selection *goquery.Selection) string {
			selector := `.SeriesName[ng-bind-html="Series.s"]`
			return selection.Find(selector).First().Text()
		},
		URL: func(selection *goquery.Selection) string {
			selector := `.SeriesName[ng-bind-html="Series.s"]`
			return selection.Find(selector).First().AttrOr("href", "")
		},
		Cover: func(selection *goquery.Selection) string {
			return selection.Find("a.SeriesName > img.img-fluid").AttrOr("src", "")
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[4]
		},
	},
	VolumeExtractor: &scraper.VolumeExtractor{
		Selector: "body",
		Number: func(selection *goquery.Selection) float32 {
			return 1.0
		},
	},
	ChapterExtractor: &scraper.ChapterExtractor{
		Selector: ".ChapterLink",
		Title: func(selection *goquery.Selection) string {
			name := selection.Find("span").First().Text()
			return name
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[4]
		},
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("href", "")
		},
		Date: func(selection *goquery.Selection) libmangal.Date {
			layout := "01/02/2006"
			publishedDate := selection.Find("span.float-right").Text()
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
		Action: func(p *rod.Page) error {
			selector := ".ShowAllChapters"
			if p.MustHas(selector) {
				element, err := p.Element(selector)
				if err != nil {
					return err
				}
				_ = element.MustClick()
				mango.Log("clicked on Show All Chapters")
			}
			return nil
		},
	},
	PageExtractor: &scraper.PageExtractor{
		Selector: ".img-fluid",
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("src", "")
		},
		// Either use the Cookie "FullPage=yes" or this action
		// Action: func(p *rod.Page) error {
		// 	selector := ".DesktopNav > div > div:nth-child(4) > button"
		// 	if p.MustHas(selector) {
		// 		element, err := p.Element(selector)
		// 		if err != nil {
		// 			return err
		// 		}
		// 		_ = element.MustClick()
		// 		mango.Log("clicked on Long Strip on nav bar")
		// 	}
		// 	return nil
		// },
	},
}
