package scraper

import (
	"time"

	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/fetch"
	"golang.org/x/time/rate"
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

	// Headers: Custom headers to pass to the request.
	Headers map[string]string

	// Fetch: How the pages are requested, direct by default (see fetch.Strategy).
	Fetch fetch.Strategy
	// LoadWait: With fetch.Headless, wait this long for the page to load instead of its load
	// event (e.g. html filled in by scripts after it).
	LoadWait time.Duration
	// LocalStorage: With fetch.Headless, local storage values to set before the requests.
	LocalStorage map[string]string
	// RateLimit: Optional, html requests per second (with RateBurst at once), instead of
	// ~4/s after a burst of 4. E.g. rate.Every(2*time.Second) for keiyoushi's rateLimit(1, 2.seconds).
	RateLimit rate.Limit
	RateBurst int
	// Middlewares: Optional hooks around every html request (headers, url rewrites, ...), the
	// first is the outermost.
	Middlewares []fetch.Middleware
	// Retry: How an html request answered 429 or 5xx is tried again, fetch.DefaultRetry if
	// nil (MaxTries 1 never retries).
	Retry *fetch.Retry

	// BaseURL: Base URL of the source.
	BaseURL string
	// GenerateSearchURL: Create search URL from the query.
	// E.g. "one piece" -> "https://manganelo.com/search/story/one_piece"
	GenerateSearchURL func(baseUrl, query string) (string, error)
	// SearchRequest: Optional, instead of GenerateSearchURL (and SearchPagination's URL): the
	// request of each page of the search (e.g. a POST).
	SearchRequest SearchRequestFunc
	// SearchPagination: Optional, for a search over several pages (one page without it).
	SearchPagination *Pagination
	// GenerateSearchByIDURL: Create search URL from the id.
	// E.g. (one piece) "manga-aa88620" -> "https://chapmanganelo.com/manga-aa88620"
	GenerateSearchByIDURL func(baseUrl, id string) (string, error)
	// ChaptersURL: Optional, the page listing the chapters if it isn't the manga page.
	ChaptersURL func(manga mango.Manga) string
	// ChaptersRequest: Optional, instead of ChaptersURL, the request listing the chapters (e.g.
	// a POST).
	ChaptersRequest ChaptersRequestFunc
	// PagesURL: Optional, the page listing the images if it isn't the chapter page.
	PagesURL func(chapter mango.Chapter) string

	// MangaByIDExtractor: Responsible for finding manga elements and extracting the data.
	//
	// Used when the id of the manga is provided and the elements need to be fetched from the
	// manga page instead of the mangas list.
	MangaByIDExtractor *MangaByIDExtractor
	// MangaExtractor: Responsible for finding manga elements and extracting the data.
	MangaExtractor *MangaExtractor
	// MetadataExtractor: Optional, the manga's metadata from its page (none without it).
	MetadataExtractor *MetadataExtractor
	// SingleVolume: The site has no volumes: every manga is one volume 1 (VolumeExtractor
	// isn't used, can be nil). When the chapters are listed on the manga page (no ChaptersURL
	// or ChaptersRequest), MangaVolumes reads them from its own fetch of the page and caches
	// them for VolumeChapters, instead of a second request.
	SingleVolume bool
	// VolumeExtractor: Responsible for finding volume elements and extracting the data.
	VolumeExtractor *VolumeExtractor
	// ChapterExtractor: Responsible for finding chapter elements and extracting the data.
	ChapterExtractor *ChapterExtractor
	// PageExtractor: Responsible for finding page elements and extracting required the data.
	PageExtractor *PageExtractor
}

// defaultMaxPages: Like the api providers' search caps (asurascans, mangadex).
const defaultMaxPages = 5

// Pagination: How to get the next pages of a search, like keiyoushi's hasNextPage + page
// number (Configuration.SearchRequest, URL, or the NextSelector's href). The search stops at a
// page without new mangas.
type Pagination struct {
	// NextSelector: Optional, an element only there when there's a next page (keiyoushi's
	// searchMangaNextPageSelector). Without URL, its href is the next page. Without it, a
	// page with mangas means there may be a next one.
	NextSelector string
	// URL: Optional, the url of page n (2, 3, ...) of the search whose first page is searchURL.
	URL func(searchURL string, page int) (string, error)
	// PageSize: Optional, a page with this many mangas may have a next one, one with fewer
	// is the last (keiyoushi's madara: mangas.size == pageSize). Wins over NextSelector.
	PageSize int
	// MaxPages: At most this many pages, default 5.
	MaxPages int
}
