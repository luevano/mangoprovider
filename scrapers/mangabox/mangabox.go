package mangabox

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
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
			Number: func(selection *goquery.Selection) int {
				return 1
			},
		},
		ChapterExtractor: &scraper.ChapterExtractor{
			Selector: "div.chapter-list div.row, ul.row-content-chapter li",
			Title: func(selection *goquery.Selection) string {
				title := selection.Find("a").Text()
				// ignore "Vol. N" from title
				if strings.HasPrefix(title, "Vol.") {
					splitted := strings.Split(title, " ")
					title = strings.Join(splitted[1:], " ")
				}
				return title
			},
			ID: func(_url string) string {
				return strings.Join(strings.Split(_url, "/")[3:], "/")
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find("a").AttrOr("href", "")
			},
			Date: func(selection *goquery.Selection) libmangal.Date {
				publishedDate := strings.TrimSpace(selection.Find(dateSelector).Text())
				date, err := time.Parse(dateLayout, publishedDate)
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
