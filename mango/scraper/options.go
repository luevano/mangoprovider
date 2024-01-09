package scraper

import "time"

// Options: Defines behavior of the scraper
type Options struct {
	// Delay between requests
	Delay time.Duration
	// Parallelism of the scraper
	Parallelism uint8

	// ReverseChapters if true, chapters will be shown in reverse order
	ReverseChapters bool

	// NeedsHeadlessBrowser if true, a headless browser will be used to proxy any request
	NeedsHeadlessBrowser bool

	// BaseURL of the source
	BaseURL string
	// GenerateSearchURL function to create search URL from the query.
	// E.g. "one piece" -> "https://manganelo.com/search/story/one%20piece"
	GenerateSearchURL func(baseUrl, query string) (string, error)

	// MangaExtractor is responsible for finding manga elements and extracting required data from them
	MangaExtractor *MangaExtractor
	// ChapterExtractor is responsible for finding chapter elements and extracting required data from them
	ChapterExtractor *ChapterExtractor
	// PageExtractor is responsible for finding page elements and extracting required data from them
	PageExtractor *PageExtractor
}
