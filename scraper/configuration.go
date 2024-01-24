package scraper

import (
	"time"

	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

// Configuration: Defines behavior of the scraper.
type Configuration struct {
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
	// Cookies: Custom cookies to pass to the request. It is a string, as it is passed as a header.
	Cookies string

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

// Get the extractor Actions.
func (c *Configuration) GetActions() map[rod.ActionType]rod.Action {
	return map[rod.ActionType]rod.Action{
		rod.ActionManga:   c.MangaExtractor.Action,
		rod.ActionVolume:  c.VolumeExtractor.Action,
		rod.ActionChapter: c.ChapterExtractor.Action,
		rod.ActionPage:    c.PageExtractor.Action,
	}
}
