package scraper

import (
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
	mango "github.com/luevano/mangoprovider"
)

func setupCollector(collector *colly.Collector, refererType string, config Configuration) error {
	collector.OnRequest(func(r *colly.Request) {
		var referer string
		switch refererType {
		case "volume":
			referer = r.Ctx.GetAny("manga").(mango.Manga).URL
		case "chapter":
			referer = r.Ctx.GetAny("volume").(mango.Volume).Manga_.URL
		case "page":
			referer = r.Ctx.GetAny("chapter").(mango.Chapter).URL
		default:
			referer = "https://google.com"
		}
		r.Headers.Set("Referer", referer)
		r.Headers.Set("accept-language", "en-US") // TODO: remove this? shouldn't specify a language
		r.Headers.Set("Accept", "text/html")
		r.Headers.Set("Host", config.BaseURL)
		r.Headers.Set("User-Agent", mango.UserAgent)
		if config.Cookies != "" {
			r.Headers.Set("Cookie", config.Cookies)
		}
	})

	err := collector.Limit(&colly.LimitRule{
		Parallelism: int(config.Parallelism),
		RandomDelay: config.Delay,
		DomainGlob:  "*",
	})
	if err != nil {
		return err
	}

	return nil
}

// TODO: refactor this function? is it needed?
func checkForRedirect(options *Configuration) error {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(options.BaseURL)
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
		options.BaseURL = loc.String()
	}
	return nil
}

// TODO: refactor these 2 functions, unnecessary abstraction?
func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func cleanName(name string) string {
	return standardizeSpaces(newLineCharacters.ReplaceAllString(name, " "))
}
