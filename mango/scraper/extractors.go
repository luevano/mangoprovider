package scraper

import (
	"time"

	"github.com/PuerkitoBio/goquery"
)

// TODO: add volume extractor and then the collector to the scraper

// MangaExtractor is responsible for finding specified elements by selector and extracting required data from them
type MangaExtractor struct {
	// Selector CSS selector
	Selector string
	// Name function to get name from element found by selector.
	Name func(*goquery.Selection) string
	// URL function to get URL from element found by selector.
	URL func(*goquery.Selection) string
	// Cover function to get cover from element found by selector. Used by manga extractor
	Cover func(*goquery.Selection) string
}

// ChapterExtractor is responsible for finding specified elements by selector and extracting required data from them
type ChapterExtractor struct {
	// Selector CSS selector
	Selector string
	// Name function to get name from element found by selector.
	Name func(*goquery.Selection) string
	// URL function to get URL from element found by selector.
	URL func(*goquery.Selection) string
	// Volume function to get volume from element found by selector. Used by chapters extractor
	Volume func(*goquery.Selection) string
	// Date function to get the published date of the chapter if available.
	Date func(*goquery.Selection) *time.Time
}

// PageExtractor is responsible for finding specified elements by selector and extracting required data from them
type PageExtractor struct {
	// Selector CSS selector
	Selector string
	// Name function to get name from element found by selector.
	Name func(*goquery.Selection) string
	// URL function to get URL from element found by selector.
	URL func(*goquery.Selection) string
}
