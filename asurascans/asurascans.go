package asurascans

import (
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/luevano/mangoprovider/mango/scraper"
)

const dateLayout = "January 2, 2006"

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-asurascans",
	Name:        "AsuraScans",
	Version:     "0.1.0",
	Description: "AsuraScans scraper",
	Website:     "https://asuracomics.com/",
}

var scraperOptions = &scraper.Options{
	Name:                 providerInfo.ID,
	Delay:                50 * time.Millisecond,
	Parallelism:          15,
	ReverseChapters:      true,
	NeedsHeadlessBrowser: true,
	BaseURL:              providerInfo.Website,
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
			return strings.TrimSpace(selection.AttrOr("title", ""))
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
		Number: func(selection *goquery.Selection) int {
			return 1
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
			date := selection.Find(".chapterdate").Text()
			t, err := time.Parse(dateLayout, date)
			if err != nil {
				// if failed to parse date, use scraping day
				t = time.Now()
			}
			return libmangal.Date{
				Year:  t.Year(),
				Month: int(t.Month()),
				Day:   t.Day(),
			}
		},
		ScanlationGroup: func(_ *goquery.Selection) string {
			return "asurascans"
		},
	},
	PageExtractor: &scraper.PageExtractor{
		Selector: "#readerarea img",
		URL: func(selection *goquery.Selection) string {
			return selection.AttrOr("src", "")
		},
	},
}

// TODO: this is generic, need to refactor
func Loader(options mango.Options) libmangal.ProviderLoader {
	s, err := scraper.NewScraper(scraperOptions, options.HeadlessOptions)
	if err != nil {
		panic(err)
	}

	return mango.ProviderLoader{
		ProviderInfo: providerInfo,
		Options:      options,
		Funcs: mango.ProviderFuncs{
			SearchMangas:   s.SearchMangas,
			MangaVolumes:   s.MangaVolumes,
			VolumeChapters: s.VolumeChapters,
			ChapterPages:   s.ChapterPages,
		},
	}
}
