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
	// LoadWait: Wait time for the page to load
	// (for headless requests where the HTML is loading dynamically for example).
	LoadWait time.Duration
	// Parallelism: Parallelism of the scraper.
	Parallelism uint8

	// ReverseChapters: If chapters should be shown in reverse order.
	ReverseChapters bool

	// NeedsHeadlessBrowser: If a headless browser should be used to proxy any request.
	NeedsHeadlessBrowser bool
	// LocalStorage: local storage values to set before the requests.
	LocalStorage map[string]string
	// Headers: Custom headers to pass to the request.
	Headers map[string]string

	// BaseURL: Base URL of the source.
	BaseURL string
	// TODO: remove unnecessary baseUrl in these generate methods
	//
	// GenerateSearchURL: Create search URL from the query.
	// E.g. "one piece" -> "https://manganelo.com/search/story/one_piece"
	GenerateSearchURL func(baseUrl, query string) (string, error)
	// GenerateSearchByIDURL: Create search URL from the id.
	// E.g. (one piece) "manga-aa88620" -> "https://chapmanganelo.com/manga-aa88620"
	GenerateSearchByIDURL func(baseUrl, id string) (string, error)

	// MangaByIDExtractor: Responsible for finding manga elements and extracting the data.
	//
	// Used when the id of the manga is provided and the elements need to be fetched from the
	// manga page instead of the mangas list.
	MangaByIDExtractor *MangaByIDExtractor
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
