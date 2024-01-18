package scraper

import "time"

// Options: Defines behavior of the scraper.
type Options struct {
	// Name: Name of the scraper. E.g. "mangapill"
	Name string
	// Delay: Delay between requests.
	Delay time.Duration
	// Parallelism: Parallelism of the scraper.
	Parallelism uint8

	// ReverseChapters: If chapters should be shown in reverse order.
	ReverseChapters bool

	// NeedsHeadlessBrowser: If a headless browser should be used to proxy any request.
	NeedsHeadlessBrowser bool

	// BaseURL: Base URL of the source.
	BaseURL string
	// GenerateSearchURL: Create search URL from the query.
	// E.g. "one piece" -> "https://manganelo.com/search/story/one%20piece"
	GenerateSearchURL func(baseUrl, query string) (string, error)

	// MangaExtractor: Responsible for finding manga elements and extracting the data.
	MangaExtractor *MangaExtractor
	// VolumeExtractor: Responsible for finding volume elements and extracting the data.
	VolumeExtractor *VolumeExtractor
	// ChapterExtractor: Responsible for finding chapter elements and extracting the data.
	ChapterExtractor *ChapterExtractor
	// PageExtractor: Responsible for finding page elements and extracting required the data.
	PageExtractor *PageExtractor
}
