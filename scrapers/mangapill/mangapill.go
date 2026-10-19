package mangapill

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangapill",
	Name:        "Mangapill",
	Version:     "0.4.6",
	Description: "Mangapill scraper",
	Website:     "https://mangapill.com/",
	Icon:        "https://mangapill.com/static/favicon/favicon-32x32.png",
}

// Config: A new configuration, built on every load.
func Config(_ *mango.Logger) *scraper.Configuration {
	return &scraper.Configuration{
		Name: Info.ID,
		// no rateLimit in keiyoushi (src/en/mangapill), the scraper's default ~4/s
		Delay:           50 * time.Millisecond,
		ReverseChapters: true,
		BaseURL:         Info.Website,
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
		// keiyoushi's: the "Next" button, ?page=N
		SearchPagination: &scraper.Pagination{
			NextSelector: "a.btn.btn-sm",
			URL: func(searchURL string, page int) (string, error) {
				u, err := url.Parse(searchURL)
				if err != nil {
					return "", err
				}
				params := u.Query()
				params.Set("page", strconv.Itoa(page))
				u.RawQuery = params.Encode()
				return u.String(), nil
			},
		},
		GenerateSearchByIDURL: func(baseUrl string, id string) (string, error) {
			return fmt.Sprintf("%smanga/%s", baseUrl, id), nil
		},
		MangaByIDExtractor: &scraper.MangaByIDExtractor{
			Selector: infoSelector,
			Title: func(selection *goquery.Selection) string {
				return selection.Find("div.flex.flex-col > div.mb-3 > h1").Text()
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("div.text-transparent > img").AttrOr("data-src", "")
			},
		},
		// keiyoushi's fetchMangaUpdate details: no authors on the site
		MetadataExtractor: &scraper.MetadataExtractor{
			IDCode: "mp",
			Extract: func(page *goquery.Selection, meta *mangadata.Metadata) {
				info := page.Find(infoSelector).First()
				meta.EnglishTitle = mango.CleanString(info.Find("div.flex.flex-col > div.mb-3 > h1").First().Text())
				meta.Summary = strings.TrimSpace(info.Find("div.flex.flex-col > div.mb-3 > p").First().Text())
				meta.CoverImage = info.Find("div.text-transparent > img").AttrOr("data-src", "")
				meta.GenreList = scraper.Texts(info.Find("a[href*=genre]"))
				info.Find("label").Each(func(_ int, label *goquery.Selection) {
					value := mango.CleanString(label.Next().Text())
					switch mango.CleanString(label.Text()) {
					case "Type":
						meta.PublicationFormat, meta.CountryOfOrigin = value, mango.CountryOfType(value)
					case "Status":
						meta.PublicationStatus = mango.ParseStatus(value)
					case "Year":
						meta.DateStart = scraper.YearDate(value)
					}
				})
			},
		},
		MangaExtractor: &scraper.MangaExtractor{
			Selector: "body > div.container.py-3 > div.my-3.grid.justify-end.gap-3.grid-cols-2.md\\:grid-cols-3.lg\\:grid-cols-5 > div",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("div a div.leading-tight").Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find("div a:first-child").AttrOr("href", "")
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("img").AttrOr("data-src", "")
			},
			// the id is in the form <number>/<manga-name> as with just <number>
			// the url returns 404 (in case it's needed)
			ID: pathID,
		},
		// a flat chapter list on the manga page (checked 2026-10-16, One Piece too)
		SingleVolume: true,
		ChapterExtractor: &scraper.ChapterExtractor{
			Selector: "div[data-filter-list] a",
			Title: func(selection *goquery.Selection) string {
				return selection.Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("href", "")
			},
			// "Group 2 Chapter 10" is another translation's chapter, kept apart as an extra
			Number: func(title string) (float32, bool) {
				if groupRegex.MatchString(title) {
					return 0, false
				}
				return mango.ParseChapterNumber(title)
			},
			// id is constructed similar to manga id above, <number>/<chapter-name>
			ID: pathID,
			ScanlationGroup: func(_ *goquery.Selection) string {
				return Info.Name
			},
		},
		PageExtractor: &scraper.PageExtractor{
			Selector: "picture img",
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("data-src", "")
			},
		},
	}
}

// infoSelector: The manga's details in its page.
const infoSelector = "div.container > div.flex.flex-col.sm\\:flex-row.my-3"

// pathID: The path after its first part, e.g. "<number>/<name>" for /manga/<number>/<name>.
var groupRegex = regexp.MustCompile(`(?i)^group\s*\d+\b`)

func pathID(_url string) string {
	if parts := scraper.PathParts(_url); len(parts) >= 2 {
		return strings.Join(parts[1:], "/")
	}
	return ""
}
