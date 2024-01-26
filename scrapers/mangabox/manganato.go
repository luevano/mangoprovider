package mangabox

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var ManganatoInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-manganato",
	Name:        "Manganato",
	Version:     "0.1.0",
	Description: "Manganato scraper",
	Website:     "https://manganato.com/",
}

var ManganatoConfig = manganato()

func manganato() *scraper.Configuration {
	m := Mangabox(ManganatoInfo.ID, ManganatoInfo.Website, "Jan 02,06")

	m.GenerateSearchURL = func(baseUrl, query string) (string, error) {
		// path is /search/story/
		// No longer required? the baseurl works just fine
		// template := "https://chapmanganato.com/" + baseUrl + "/search/story/%s"
		// return fmt.Sprintf(template, query), nil
		query = strings.ReplaceAll(query, " ", "_")
		u, _ := url.Parse(baseUrl)
		u.Path = fmt.Sprintf("/search/story/%s", query)

		return u.String(), nil
	}
	m.MangaExtractor.Selector = "div.search-story-item"
	m.MangaExtractor.ID = func(_url string) string {
		return strings.Split(_url, "/")[3]
	}

	return m
}
