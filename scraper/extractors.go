package scraper

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

// MangaByIDExtractor: responsible for finding manga elements by selector and extracting the data.
//
// Used when the id of the manga is provided and the elements need to be fetched from the
// manga page instead of the mangas list.
type MangaByIDExtractor struct {
	// Selector: CSS selector.
	Selector string
	// Title: Get title from element found by selector.
	Title func(*goquery.Selection) string
	// Cover: Get cover from element found by selector.
	Cover func(*goquery.Selection) string
}

// MangaExtractor: responsible for finding manga elements by selector and extracting the data.
type MangaExtractor struct {
	// Selector: CSS selector.
	Selector string
	// Title: Get title from element found by selector.
	Title func(*goquery.Selection) string
	// URL: Get URL from element found by selector.
	URL func(*goquery.Selection) string
	// ID: Get id from parsed url string, empty skips the manga (e.g. an odd url).
	ID func(string) string
	// Cover: Get cover from element found by selector.
	Cover func(*goquery.Selection) string
	// Action: With fetch.Headless, something to run on the page once loaded.
	Action rod.Action
}

// VolumeExtractor: responsible for finding volume elements by selector and extracting the data.
type VolumeExtractor struct {
	// Selector: CSS selector.
	Selector string
	// Number: Get number from element found by selector.
	Number func(*goquery.Selection) float32
	// Action: With fetch.Headless, something to run on the page once loaded.
	Action rod.Action
}

// ChapterExtractor: responsible for finding chapter elements by selector and extracting the data.
type ChapterExtractor struct {
	// Selector: CSS selector.
	Selector string
	// Title: Get title from element found by selector.
	Title func(*goquery.Selection) string
	// URL: Get URL from element found by selector.
	URL func(*goquery.Selection) string
	// ID: Get id from parsed url string, empty skips the chapter (e.g. an odd url).
	ID func(string) string
	// Number: Optional, get the number from the title, mango.ParseChapterNumber if nil.
	// Not ok means the chapter has none, it goes right after the previous one.
	Number func(title string) (number float32, ok bool)
	// Date: Get the published date of the chapter if available.
	Date func(*goquery.Selection) metadata.Date
	// ScanlationGroup: Get the scanlation group if available.
	ScanlationGroup func(*goquery.Selection) string
	// Action: With fetch.Headless, something to run on the page once loaded.
	Action rod.Action
}

// PageExtractor: responsible for finding page elements by selector and extracting the data.
type PageExtractor struct {
	// Selector: CSS selector.
	Selector string
	// URL: Get URL from element found by selector.
	URL func(*goquery.Selection) string
	// URLs: Get all URLs from element found by selector.
	URLs func(*goquery.Selection) []string
	// Action: With fetch.Headless, something to run on the page once loaded.
	Action rod.Action
}

// PathParts: The parts of the url's path, e.g. ["manga", "1", "one-piece"] for
// https://site/manga/1/one-piece/, none if it doesn't parse.
func PathParts(rawURL string) []string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	p := strings.Trim(u.Path, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}
