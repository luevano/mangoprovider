package mangaplus

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/google/uuid"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangaplus",
	Name:        "MangaPlus",
	Version:     "0.1.0",
	Description: "MangaPlus scraper",
	Website:     "https://mangaplus.shueisha.co.jp/",
}

// TODO: support other languages for localstorage
var Config = &scraper.Configuration{
	Name:                 Info.ID,
	Delay:                50 * time.Millisecond,
	NeedsHeadlessBrowser: true,
	BaseURL:              Info.Website,
	LocalStorage: map[string]string{
		"contentsV2": "['en']",
		"service":    "en",
		"quarity":    "super_high",
	},
	Headers: map[string]string{
		"SESSION-TOKEN": func() string {
			randUUID, err := uuid.NewRandom()
			if err != nil {
				return ""
			}
			return randUUID.String()
		}(),
	},
	GenerateSearchURL: func(baseUrl string, query string) (string, error) {
		// path is /search_result?keyword=<query>
		params := url.Values{}
		params.Set("keyword", query)

		u, _ := url.Parse(baseUrl)
		u.Path = "/search_result"
		u.RawQuery = params.Encode()

		return u.String(), nil
	},
	MangaExtractor: &scraper.MangaExtractor{
		Selector: "div[class^=styles-module_allTitles] > a[class^=AllTitle-module_allTitle]",
		Title: func(selection *goquery.Selection) string {
			return selection.Find("p[class^=AllTitle-module_title]").Text()
		},
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("href", "")
		},
		Cover: func(selection *goquery.Selection) string {
			return selection.Find("img[class^=AllTitle-module_image]").AttrOr("data-src", "")
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[4]
		},
	},
	VolumeExtractor: &scraper.VolumeExtractor{
		Selector: "div[class^=ChapterList-module_chapterListTitleWrapper]",
		Number: func(selection *goquery.Selection) float32 {
			return 1.0
		},
	},
	ChapterExtractor: &scraper.ChapterExtractor{
		Selector: "div[class^=ChapterListItem-module_chapterListItem]",
		Title: func(selection *goquery.Selection) string {
			return selection.Find("p[class^=ChapterListItem-module_title]").Text()
		},
		URL: func(selection *goquery.Selection) string {
			// There is no direct URL that can be selected,
			// but each chapter has a link to its comments and the id is the same
			commentURL := selection.Find("a[class^=ChapterListItem-module_commentContainer]").AttrOr("href", "")
			chapterID := strings.Split(commentURL, "/")[2]
			return fmt.Sprintf("/viewer/%s", chapterID)
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[4]
		},
		Date: func(selection *goquery.Selection) libmangal.Date {
			layout := "Jan 2, 2006"
			publishedDate := selection.Find("p[class^=ChapterListItem-module_date]").Text()
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
		// Selector: "div.zao-surface > div.zao-container > div.zao-image-container",
		Selector: "div.zao-image-container",
		URL: func(selection *goquery.Selection) string {
			rawURL := selection.Find("img.zao-image").AttrOr("src", "")
			return strings.Replace(rawURL, "blob:", "", 1)
		},
		Action: func(p *rod.Page) error {
			// TODO: need to control this better,
			// this feels like a good default so far
			totalScrolls := 200

			selector := "p[class^=Viewer-module_pageNumber] > span"
			if p.MustHas(selector) {
				element := p.MustElement(selector)
				pageCountS := element.MustText()
				pageCountS = strings.TrimSpace(strings.Replace(pageCountS, "/", "", 1))
				pageCount, err := strconv.Atoi(pageCountS)
				if err != nil {
					return err
				}
				totalScrolls = 2 * pageCount
			}

			kb := p.MustActivate().Keyboard
			count := 0
			for count < totalScrolls {
				kb.MustType(input.ArrowDown)
				// TODO: remove or tweak this delay
				time.Sleep(50 * time.Millisecond)
				count += 1
			}
			return nil
		},
	},
}
