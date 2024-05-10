package asurascans

import (
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

// TODO: add extra option for extracting chapter number (solo leveling is wrong)

var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-asurascans",
	Name:        "AsuraScans",
	Version:     "0.1.1",
	Description: "AsuraScans scraper",
	Website:     "https://asuracomics.com/",
}

var Config = &scraper.Configuration{
	Name:                 Info.ID,
	Delay:                50 * time.Millisecond,
	ReverseChapters:      true,
	NeedsHeadlessBrowser: true, // TODO: does it really need it?
	BaseURL:              Info.Website,
	GenerateSearchURL: func(baseUrl string, query string) (string, error) {
		// path is /?s=
		params := url.Values{}
		params.Set("s", query)
		u, _ := url.Parse(baseUrl)
		u.Path = "/"
		u.RawQuery = params.Encode()

		return u.String(), nil
	},
	MangaExtractor: &scraper.MangaExtractor{
		Selector: ".bsx > a",
		Title: func(selection *goquery.Selection) string {
			return selection.AttrOr("title", "")
		},
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("href", "")
		},
		Cover: func(selection *goquery.Selection) string {
			return selection.Find("img").AttrOr("src", "")
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[4]
		},
	},
	VolumeExtractor: &scraper.VolumeExtractor{
		// selector that points to only 1 element ("Chapter MangaName" header)
		Selector: "body > div > div.wrapper > div.postbody > article.hentry > div.bixbox.bxcl.epcheck > div.releases > h2",
		Number: func(selection *goquery.Selection) float32 {
			return 1.0
		},
		// AsuraScans doesn't really provide volumes, some chapters have "Vol." prefix, need to figure out how to implement this as this was used inside the chapter extractor on original mangal
		// Volume: func(selection *goquery.Selection) string {
		// 	name := selection.Find(".chapternum").Text()
		// 	if strings.HasPrefix(name, "Vol.") {
		// 		splitted := strings.Split(name, " ")
		// 		return splitted[0]
		// 	}
		// 	return ""
		// },
	},
	ChapterExtractor: &scraper.ChapterExtractor{
		Selector: "#chapterlist > ul li",
		Title: func(selection *goquery.Selection) string {
			name := selection.Find(".chapternum").Text()
			return name
		},
		ID: func(_url string) string {
			return strings.Split(_url, "/")[3]
		},
		URL: func(selection *goquery.Selection) string {
			return selection.Find("a").AttrOr("href", "")
		},
		Date: func(selection *goquery.Selection) libmangal.Date {
			layout := "January 2, 2006"
			publishedDate := selection.Find(".chapterdate").Text()
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
		Selector: "#readerarea img",
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("src", "")
		},
	},
}
