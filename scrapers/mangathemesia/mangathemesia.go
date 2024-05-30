package mangathemesia

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
// TODO: add extra option for extracting chapter number
// (solo leveling is wrong, at least for flamecomics)

// Mangathemesia is a generic type of website used by Asurascans, Flamecomics, etc.
func Mangathemesia(name, baseUrl, mangaDir string) *scraper.Configuration {
	// Looks like they don't need a headless browser (and thus no load wait), but will keep an eye
	return &scraper.Configuration{
		// LoadWait:        1 * time.Second,
		// NeedsHeadlessBrowser: true,
		Name:            name,
		Delay:           50 * time.Millisecond,
		ReverseChapters: true,
		BaseURL:         baseUrl,
		GenerateSearchURL: func(baseUrl string, query string) (string, error) {
			// path is /?s=
			params := url.Values{}
			params.Set("s", query)
			u, _ := url.Parse(baseUrl)
			u.Path = "/"
			u.RawQuery = params.Encode()

			return u.String(), nil
		},
		GenerateSearchByIDURL: func(baseUrl string, id string) (string, error) {
			dir := strings.Trim(mangaDir, "/")
			return fmt.Sprintf("%s%s/%s", baseUrl, dir, id), nil
		},
		MangaByIDExtractor: &scraper.MangaByIDExtractor{
			Selector: "div.bigcontent, div.animefull, div.main-info, div.postbody",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("h1.entry-title, .ts-breadcrumb li:last-child span").First().Text()
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find(".infomanga > div[itemprop=image] img, .thumb img").AttrOr("src", "")
			},
		},
		MangaExtractor: &scraper.MangaExtractor{
			Selector: ".utao .uta .imgu, .listupd .bs .bsx, .listo .bs .bsx",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("a").AttrOr("title", "")
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find("a").AttrOr("href", "")
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("img").AttrOr("src", "")
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
			Selector: "div.bxcl li, div.cl li, #chapterlist li, ul li:has(div.chbox):has(div.eph-num)",
			Title: func(selection *goquery.Selection) string {
				return selection.Find(".lch a, .chapternum").Text()
			},
			ID: func(_url string) string {
				return strings.Split(_url, "/")[3]
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find("a").AttrOr("href", "")
			},
			Date: func(selection *goquery.Selection) metadata.Date {
				publishedDate := selection.Find(".chapterdate").Text()
				date, err := time.Parse("January 2, 2006", publishedDate)
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
			// Selector: "div#readerarea img",
			// Taken from tachiyomi (RIP) source, but old one should also work fine
			Selector: "div.rdminimal > img, div.rdminimal > p > img, div.rdminimal > a > img, div.rdminimal > p > a > img, div.rdminimal > noscript > img, div.rdminimal > p > noscript > img, div.rdminimal > a > noscript > img, div.rdminimal > p > a > noscript > img",
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("src", "")
			},
		},
	}
}
