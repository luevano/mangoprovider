package toonily

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-toonily",
	Name:        "Toonily",
	Version:     "0.1.4",
	Description: "Toonily scraper",
	Website:     "https://toonily.com/",
}

// Config: A new configuration, built on every load.
func Config(_ *mango.Logger) *scraper.Configuration {
	return &scraper.Configuration{
		Name: Info.ID,
		// no rateLimit in keiyoushi (src/en/toonily, lib-multisrc/madara), the scraper's default ~4/s
		Delay:           50 * time.Millisecond,
		ReverseChapters: true,
		BaseURL:         Info.Website,
		Headers: map[string]string{
			"Cookie": "toonily-mature=1",
		},
		GenerateSearchURL: func(baseUrl string, query string) (string, error) {
			// path is /search/
			u, _ := url.Parse(baseUrl)
			query = strings.ReplaceAll(query, " ", "-")
			u.Path = fmt.Sprintf("/search/%s", query)

			return u.String(), nil
		},
		// madara's next page selector (keiyoushi before Madara 1.6), /page/N/
		SearchPagination: &scraper.Pagination{
			NextSelector: "div.nav-previous, nav.navigation-ajax, a.nextpostslink",
			URL: func(searchURL string, page int) (string, error) {
				return strings.TrimSuffix(searchURL, "/") + "/page/" + strconv.Itoa(page) + "/", nil
			},
		},
		GenerateSearchByIDURL: func(baseUrl string, id string) (string, error) {
			return fmt.Sprintf("%swebtoon/%s", baseUrl, id), nil
		},
		MangaByIDExtractor: &scraper.MangaByIDExtractor{
			Selector: "div.tab-summary",
			Title: func(selection *goquery.Selection) string {
				titleSelection := selection.Find("div.post-content > div.post-title")
				titleSelection.Children().Find("h1 > span.manga-title-badges").Remove()
				return titleSelection.Find("h1").Text()
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("div.summary_image > a > img").AttrOr("data-src", "")
			},
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
			// e.g. https://toonily.com/webtoon/<slug>/
			ID: func(_url string) string {
				if parts := scraper.PathParts(_url); len(parts) >= 2 {
					return parts[1]
				}
				return ""
			},
		},
		VolumeExtractor: &scraper.VolumeExtractor{
			// selector that points to only 1 element
			Selector: "body",
			Number: func(selection *goquery.Selection) float32 {
				return 1.0
			},
		},
		ChapterExtractor: &scraper.ChapterExtractor{
			Selector: "div.listing-chapters_wrap > ul li",
			Title: func(selection *goquery.Selection) string {
				name := selection.Find("a").Text()
				return name
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find("a").AttrOr("href", "")
			},
			// e.g. https://toonily.com/webtoon/<slug>/chapter-1/
			ID: func(_url string) string {
				if parts := scraper.PathParts(_url); len(parts) >= 3 {
					return parts[2]
				}
				return ""
			},
			Date: func(selection *goquery.Selection) metadata.Date {
				return releaseDate(selection, time.Now())
			},
			ScanlationGroup: func(_ *goquery.Selection) string {
				return Info.Name
			},
		},
		PageExtractor: &scraper.PageExtractor{
			Selector: "div.reading-content div.page-break.no-gaps img",
			// lazy loaded images had the url in data-src, now it's src (like keiyoushi's madara)
			URL: func(selection *goquery.Selection) string {
				for _, attr := range []string{"data-src", "data-lazy-src", "src"} {
					if url := strings.TrimSpace(selection.AttrOr(attr, "")); url != "" {
						return url
					}
				}
				return ""
			},
		},
	}
}

// releaseDate: The chapter's date, new ones say how long ago (e.g. "2 hours ago"), none if it's
// neither.
func releaseDate(selection *goquery.Selection, now time.Time) metadata.Date {
	release := selection.Find("span.chapter-release-date")
	text := release.Find("i").Text()
	if title := release.Find("a").AttrOr("title", ""); title != "" {
		text = title
	}
	date, _ := mango.ParseDate(text, now, "Jan 2, 06")
	return date
}
