package weebcentral

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
	"golang.org/x/time/rate"
)

// searchPageSize: Results in a search page.
const searchPageSize = 32

// MangaSee shut down and moved to Weeb Central; its id is in the history so it still resolves.
var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-weebcentral",
	Name:        "Weeb Central",
	Version:     "0.4.3",
	Description: "Weeb Central scraper",
	Website:     "https://weebcentral.com/",
	History: []libmangal.ProviderAlias{
		{ID: mango.BundleID + "-mangasee", Name: "MangaSee", Website: "https://mangasee123.com/"},
	},
}

// Config: A new configuration, built on every load.
func Config(log *mango.Logger) *scraper.Configuration {
	return &scraper.Configuration{
		Name:            Info.ID,
		Delay:           50 * time.Millisecond,
		ReverseChapters: true,
		// keiyoushi's rateLimit(1, 2.seconds) on the site's host (src/en/weebcentral/.../WeebCentral.kt),
		// images are on other hosts
		RateLimit: rate.Every(2 * time.Second),
		RateBurst: 1,
		BaseURL:   Info.Website,
		GenerateSearchURL: func(baseUrl string, query string) (string, error) {
			params := url.Values{}
			params.Set("text", query)
			params.Set("sort", "Best Match")
			params.Set("order", "Descending")
			params.Set("official", "Any")
			params.Set("anime", "Any")
			params.Set("adult", "Any")
			params.Set("display_mode", "Full Display")
			return baseUrl + "search/data?" + params.Encode(), nil
		},
		// keiyoushi's: 32 per page (whatever limit says), a "View More Results..." button
		// when there are more
		SearchPagination: &scraper.Pagination{
			NextSelector: "button",
			URL: func(searchURL string, page int) (string, error) {
				u, err := url.Parse(searchURL)
				if err != nil {
					return "", err
				}
				params := u.Query()
				params.Set("limit", strconv.Itoa(searchPageSize))
				params.Set("offset", strconv.Itoa((page-1)*searchPageSize))
				u.RawQuery = params.Encode()
				return u.String(), nil
			},
		},
		GenerateSearchByIDURL: func(baseUrl string, id string) (string, error) {
			return baseUrl + "series/" + id, nil
		},
		// the series page only shows the latest chapters
		ChaptersURL: func(manga mango.Manga) string {
			return Info.Website + "series/" + manga.ID + "/full-chapter-list"
		},
		// the chapter page loads its images from here
		PagesURL: func(chapter mango.Chapter) string {
			return Info.Website + "chapters/" + chapter.ID + "/images?is_prev=False&current_page=1&reading_style=long_strip"
		},
		MangaByIDExtractor: &scraper.MangaByIDExtractor{
			Selector: "main",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("h1").First().Text()
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("picture img").First().AttrOr("src", "")
			},
		},
		MangaExtractor: &scraper.MangaExtractor{
			Selector: "article.bg-base-300",
			Title: func(selection *goquery.Selection) string {
				return selection.Find(`section:nth-child(2) a[href*="/series/"]`).First().Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find(`section:nth-child(2) a[href*="/series/"]`).First().AttrOr("href", "")
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("picture img").First().AttrOr("src", "")
			},
			// e.g. https://weebcentral.com/series/01J76XY7EF75DJNQCV04HTPDZK/Berserk
			ID: func(_url string) string {
				id, _ := pathAfter(_url, "series")
				return id
			},
		},
		VolumeExtractor: &scraper.VolumeExtractor{
			Selector: "body",
			Number: func(selection *goquery.Selection) float32 {
				return 1.0
			},
		},
		ChapterExtractor: &scraper.ChapterExtractor{
			Selector: `a[href*="/chapters/"]`,
			Title: func(selection *goquery.Selection) string {
				return selection.Find("span.grow > span").First().Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("href", "")
			},
			// e.g. https://weebcentral.com/chapters/01KX4BFP5E3FEZ50TRS752CT9V
			ID: func(_url string) string {
				id, _ := pathAfter(_url, "chapters")
				return id
			},
			Number: chapterNumber,
			Date: func(selection *goquery.Selection) metadata.Date {
				date, err := time.Parse(time.RFC3339, selection.Find("time").AttrOr("datetime", ""))
				if err != nil {
					log.Log("chapter %q has no valid date", mango.CleanString(selection.Text()))
					return metadata.Date{}
				}
				return mango.DateOf(date)
			},
			ScanlationGroup: func(_ *goquery.Selection) string {
				return "Weeb Central"
			},
		},
		PageExtractor: &scraper.PageExtractor{
			Selector: `img[alt^="Page"]`,
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("src", "")
			},
		},
	}
}

// chapterNumber: Prologues are numbered before chapter 1, e.g. "Prologue 3" is 0.03.
func chapterNumber(title string) (float32, bool) {
	if n, ok := strings.CutPrefix(title, "Prologue "); ok {
		if parsed, err := strconv.ParseFloat(n, 32); err == nil && parsed < 100 {
			return float32(parsed / 100), true
		}
	}
	return mango.ParseChapterNumber(title)
}

// pathAfter: The path segment after the given one, e.g. the id after "series".
func pathAfter(_url, segment string) (string, bool) {
	parts := scraper.PathParts(_url)
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == segment {
			return parts[i+1], true
		}
	}
	return "", false
}
