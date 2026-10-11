package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

// Sets request headers via OnRequest callback for the collector.
func (s *Scraper) setCollectorOnRequest(collector *colly.Collector, collectorType rod.ActionType) {
	collector.OnRequest(func(r *colly.Request) {
		var referer string
		switch collectorType {
		case rod.ActionVolume:
			referer = r.Ctx.GetAny("manga").(mango.Manga).URL
		case rod.ActionChapter:
			referer = r.Ctx.GetAny("volume").(mango.Volume).Manga_.URL
		case rod.ActionPage:
			referer = r.Ctx.GetAny("chapter").(mango.Chapter).URL
		default:
			referer = s.config.BaseURL
		}
		r.Headers.Set("Referer", referer)
		r.Headers.Set("Accept-Language", "en-US") // TODO: remove this? shouldn't specify a language
		r.Headers.Set("Accept", "text/html")
		r.Headers.Set("Host", s.config.BaseURL) // TODO: remove this? even rod breaks when setting it
		r.Headers.Set("User-Agent", s.options.UserAgent)

		// Custom scraper headers (like "Cookie" for some sites)
		for k, v := range s.config.Headers {
			r.Headers.Set(k, v)
		}

		// Used to call the corresponding rod.Action.
		r.Headers.Set(rod.ActionTypeHeader, string(collectorType))
	})
}

// Checks redirections and sets the new BaseURL if needed.
func setBaseURLOnRedirect(ctx context.Context, config *Configuration, httpClient *http.Client) error {
	// same client (timeout, transport), just without following redirects
	client := *httpClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.BaseURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("check %s for redirects: %w", config.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		loc, err := resp.Location()
		if err != nil {
			return err
		}
		config.BaseURL = loc.String()
	}
	return nil
}

const scrapeErrorKey = "scrape_error"

// scrape: Requests the url with the collector and waits for its callbacks to finish.
//
// The collector is async, so colly only passes request errors (including 4xx/5xx) to OnError,
// they're collected here and returned along with panics in the callbacks (see recoverScrape).
func scrape(ctx context.Context, collector *colly.Collector, url string, data *colly.Context) error {
	collector.Context = ctx
	collector.OnError(func(r *colly.Response, err error) {
		switch {
		case r.StatusCode == http.StatusNotFound:
			err = fmt.Errorf("%d %w", r.StatusCode, mango.ErrNotFound)
		case r.StatusCode != 0:
			err = fmt.Errorf("%d %s", r.StatusCode, strings.ToLower(http.StatusText(r.StatusCode)))
		}
		setScrapeError(r.Ctx, fmt.Errorf("GET %s: %w", r.Request.URL, err))
	})

	if err := collector.Request(http.MethodGet, url, nil, data, nil); err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	collector.Wait()

	if err := ctx.Err(); err != nil {
		return err
	}
	if err, ok := data.GetAny(scrapeErrorKey).(error); ok {
		return err
	}
	return nil
}

// recoverScrape: Turns a panic in a collector callback (e.g. the site changed its markup) into
// an error returned by scrape, instead of crashing the whole program. Use it deferred.
func recoverScrape(e *colly.HTMLElement) {
	if r := recover(); r != nil {
		setScrapeError(e.Request.Ctx, fmt.Errorf("parse %s: %v", e.Request.URL, r))
	}
}

// setScrapeError: Keeps the first error only.
func setScrapeError(data *colly.Context, err error) {
	if data.GetAny(scrapeErrorKey) == nil {
		data.Put(scrapeErrorKey, err)
	}
}
