package manganelo

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

// TODO: check the website url, there appears to be multiple of them

var ProviderInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-manganelo",
	Name:        "Manganelo",
	Version:     "0.1.0",
	Description: "Manganelo scraper",
	Website:     "https://ww7.manganelo.tv/",
}

var Options = &scraper.Options{
	Name:            ProviderInfo.ID,
	Delay:           50 * time.Millisecond,
	Parallelism:     15,
	ReverseChapters: true,
	BaseURL:         ProviderInfo.Website,
	GenerateSearchURL: func(baseUrl string, query string) (string, error) {
		// path is /search/
		u, _ := url.Parse(baseUrl)
		u.Path = fmt.Sprintf("/search/%s", query)

		return u.String(), nil
	},
	MangaExtractor: &scraper.MangaExtractor{
		Selector: "div.search-story-item",
		Title: func(selection *goquery.Selection) string {
			return strings.TrimSpace(selection.Find("a.item-title").Text())
		},
		URL: func(selection *goquery.Selection) string {
			return selection.Find("a.item-title").AttrOr("href", "")
		},
		Cover: func(selection *goquery.Selection) string {
			return selection.Find("a.item-img > img").AttrOr("src", "")
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[4]
		},
	},
	VolumeExtractor: &scraper.VolumeExtractor{
		// selector that points to only 1 element ("Chapter name" header)
		Selector: "body > div.body-site > div.container.container-main > div.container-main-left > div.panel-story-chapter-list > p.row-title-chapter > span.row-title-chapter-name",
		Number: func(selection *goquery.Selection) int {
			return 1
		},
		// Manganelo doesn't really provide volumes, some chapters have "Vol." prefix, need to figure out how to implement this as this was used inside the chapter extractor on original mangal
		// Volume: func(selection *goquery.Selection) string {
		// 	name := selection.Find(".chapter-name").Text()
		// 	if strings.HasPrefix(name, "Vol.") {
		// 		splitted := strings.Split(name, " ")
		// 		return splitted[0]
		// 	}
		// 	return ""
		// },
	},
	ChapterExtractor: &scraper.ChapterExtractor{
		Selector: "li.a-h",
		Title: func(selection *goquery.Selection) string {
			name := selection.Find(".chapter-name").Text()
			// ignore "Vol. N" from title
			if strings.HasPrefix(name, "Vol.") {
				splitted := strings.Split(name, " ")
				name = strings.Join(splitted[1:], " ")
			}
			return name
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[4]
		},
		URL: func(selection *goquery.Selection) string {
			return selection.Find(".chapter-name").AttrOr("href", "")
		},
		Date: func(selection *goquery.Selection) libmangal.Date {
			layout := "Jan 02,06"
			publishedDate := strings.TrimSpace(selection.Find(".chapter-time").Text())
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
			return ProviderInfo.Name
		},
	},
	PageExtractor: &scraper.PageExtractor{
		Selector: ".container-chapter-reader img",
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("data-src", "")
		},
	},
}
