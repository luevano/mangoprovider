package scraper

import (
	"net/url"

	"github.com/PuerkitoBio/goquery"
	mango "github.com/luevano/mangoprovider"
)

// Request: An html request, a GET of URL unless it has a Form (e.g. madara's ajax).
type Request struct {
	URL string
	// Form: Optional, POSTed as application/x-www-form-urlencoded.
	Form url.Values
	// Headers: Optional, set on top of the scraper's (e.g. X-Requested-With).
	Headers map[string]string
}

// key: What identifies the request, e.g. for the cache.
func (r Request) key() string {
	if r.Form == nil {
		return r.URL
	}
	return "POST " + r.URL + " " + r.Form.Encode()
}

// SearchRequestFunc: The request of page n (from 1) of the search for query, see
// Configuration.SearchRequest.
type SearchRequestFunc func(baseURL, query string, page int) (Request, error)

// ChaptersRequestFunc: The request listing the manga's chapters, see
// Configuration.ChaptersRequest. mangaPage gets the manga's page, for what's only there (e.g. a
// post id).
type ChaptersRequestFunc func(manga mango.Manga, mangaPage func() (*goquery.Selection, error)) (Request, error)
