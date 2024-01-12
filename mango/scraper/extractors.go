package scraper

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
)

// MangaExtractor: responsible for finding manga elements by selector and extracting the data.
type MangaExtractor struct {
	// Selector: CSS selector
	Selector string
	// Title: Get title from element found by selector.
	Title func(*goquery.Selection) string
	// URL: Get URL from element found by selector.
	URL func(*goquery.Selection) string
	// TODO: change this to a more generic? Not sure if this is enough
	// ID: Get id from parsed url string.
	ID func(string) string
	// Cover: Get cover from element found by selector.
	Cover func(*goquery.Selection) string
}

// VolumeExtractor: responsible for finding volume elements by selector and extracting the data.
type VolumeExtractor struct {
	// Selector: CSS selector.
	Selector string
	// Number: Get number from element found by selector.
	Number func(*goquery.Selection) int
}

// ChapterExtractor: responsible for finding chapter elements by selector and extracting the data.
type ChapterExtractor struct {
	// Selector: CSS selector.
	Selector string
	// Title: Get title from element found by selector.
	Title func(*goquery.Selection) string
	// TODO: change this to a more generic? Not sure if this is enough
	// ID: Get id from parsed url string.
	ID func(string) string
	// URL: Get URL from element found by selector.
	URL func(*goquery.Selection) string
	// Date: Get the published date of the chapter if available.
	Date func(*goquery.Selection) libmangal.Date
	// ScanlationGroups: Get the scanlation groups if available.
	ScanlationGroups func(*goquery.Selection) []string
}

// PageExtractor: responsible for finding page elements by selector and extracting the data.
type PageExtractor struct {
	// Selector: CSS selector.
	Selector string
	// URL: Get URL from element found by selector.
	URL func(*goquery.Selection) string
}
