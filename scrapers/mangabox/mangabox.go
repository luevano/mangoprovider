package mangabox

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangoprovider/scraper"
)

// TODO: add manga search pager?

// Mangabox is a generic type of website used by Manganato, Manganelo, Mangakakalot, etc.
func Mangabox(name, baseUrl, searchPath, dateLayout, dateSelector string) *scraper.Configuration {
	return &scraper.Configuration{
		Name:            name,
		Delay:           50 * time.Millisecond,
		ReverseChapters: true,
		BaseURL:         baseUrl,
		GenerateSearchURL: func(baseUrl string, query string) (string, error) {
			query = strings.ReplaceAll(query, " ", "_")
			u, _ := url.Parse(baseUrl)
			u.Path = fmt.Sprintf(searchPath, query)

			return u.String(), nil
		},
		GenerateSearchByIDURL: func(baseUrl string, id string) (string, error) {
			return fmt.Sprintf("%s%s", baseUrl, id), nil
		},
		MangaByIDExtractor: &scraper.MangaByIDExtractor{
			Selector: "div.manga-info-top, div.panel-story-info",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("h1, h2").First().Text()
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("div.manga-info-pic img, span.info-image img").AttrOr("src", "")
			},
		},
		MangaExtractor: &scraper.MangaExtractor{
			Selector: ".panel-search-story .search-story-item",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("a.item-title").Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find("a.item-title").AttrOr("href", "")
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("a img").AttrOr("src", "")
			},
			ID: func(_url string) string {
				return strings.Split(_url, "/")[3]
			},
		},
		// TODO: Parse "Vol. #"?
		VolumeExtractor: &scraper.VolumeExtractor{
			Selector: "body",
			Number: func(selection *goquery.Selection) float32 {
				return 1.0
			},
		},
		ChapterExtractor: &scraper.ChapterExtractor{
			Selector: "div.chapter-list div.row, ul.row-content-chapter li",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("a").Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find("a").AttrOr("href", "")
			},
			ID: func(_url string) string {
				return strings.Join(strings.Split(_url, "/")[3:], "/")
			},
			Date: func(selection *goquery.Selection) metadata.Date {
				publishedDate := strings.TrimSpace(selection.Find(dateSelector).Text())
				date, err := time.Parse(dateLayout, publishedDate)
				if err != nil {
					// if failed to parse date, use scraping day
					date = time.Now()
				}
				return metadata.Date{
					Year:  date.Year(),
					Month: int(date.Month()),
					Day:   date.Day(),
				}
			},
			ScanlationGroup: func(_ *goquery.Selection) string {
				return name
			},
		},
		PageExtractor: &scraper.PageExtractor{
			Selector: "div#vungdoc img, div.container-chapter-reader img",
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("src", "")
			},
		},
	}
}
