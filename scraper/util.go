package scraper

import (
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

// Sets request headers via OnRequest callback for the collector.
func setCollectorOnRequest(collector *colly.Collector, config *Configuration, collectorType rod.ActionType) {
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
			referer = "https://google.com"
		}
		r.Headers.Set("Referer", referer)
		r.Headers.Set("Accept-Language", "en-US") // TODO: remove this? shouldn't specify a language
		r.Headers.Set("Accept", "text/html")
		r.Headers.Set("Host", config.BaseURL) // TODO: remove this? even rod breaks when setting it
		r.Headers.Set("User-Agent", mango.UserAgent)

		// Custom scraper headers (like "Cookie" for some sites)
		for k, v := range config.Headers {
			r.Headers.Set(k, v)
		}

		// Used to call the corresponding rod.Action.
		r.Headers.Set(rod.ActionTypeHeader, string(collectorType))
	})
}

// Checks redirections and sets the new BaseURL if needed.
func setBaseURLOnRedirect(config *Configuration) error {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(config.BaseURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	defer client.CloseIdleConnections()

	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		loc, err := resp.Location()
		if err != nil {
			return err
		}
		config.BaseURL = loc.String()
	}
	return nil
}

// Returns the string with single spaces. E.g. "    " -> " "
func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Get the string with all whitespace standardized.
func cleanString(s string) string {
	return standardizeSpaces(newLineCharacters.ReplaceAllString(s, " "))
}
