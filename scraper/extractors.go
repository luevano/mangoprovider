package scraper

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

// MangaExtractor: responsible for finding manga elements by selector and extracting the data.
type MangaExtractor struct {
	// Selector: CSS selector
	Selector string
	// Title: Get title from element found by selector.
	Title func(*goquery.Selection) string
	// URL: Get URL from element found by selector.
	URL func(*goquery.Selection) string
	// ID: Get id from parsed url string.
	ID func(string) string
	// Cover: Get cover from element found by selector.
	Cover func(*goquery.Selection) string
	// Action: Something to execute on a headless browser after page is loaded.
	Action rod.Action
}

// VolumeExtractor: responsible for finding volume elements by selector and extracting the data.
type VolumeExtractor struct {
	// Selector: CSS selector.
	Selector string
	// Number: Get number from element found by selector.
	Number func(*goquery.Selection) int
	// Action: Something to execute on a headless browser after page is loaded.
	Action rod.Action
}

// ChapterExtractor: responsible for finding chapter elements by selector and extracting the data.
type ChapterExtractor struct {
	// Selector: CSS selector.
	Selector string
	// Title: Get title from element found by selector.
	Title func(*goquery.Selection) string
	// ID: Get id from parsed url string.
	ID func(string) string
	// URL: Get URL from element found by selector.
	URL func(*goquery.Selection) string
	// Date: Get the published date of the chapter if available.
	Date func(*goquery.Selection) libmangal.Date
	// ScanlationGroup: Get the scanlation group if available.
	ScanlationGroup func(*goquery.Selection) string
	// Action: Something to execute on a headless browser after page is loaded.
	Action rod.Action
}

// PageExtractor: responsible for finding page elements by selector and extracting the data.
type PageExtractor struct {
	// Selector: CSS selector.
	Selector string
	// URL: Get URL from element found by selector.
	URL func(*goquery.Selection) string
	// Action: Something to execute on a headless browser after page is loaded.
	Action rod.Action
}
