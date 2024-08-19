package mangabox

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var MangakakalotInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangakakalot",
	Name:        "Mangakakalot",
	Version:     "0.2.1",
	Description: "Mangakakalot scraper",
	Website:     "https://mangakakalot.com/",
}

var MangakakalotConfig = mangakakalot()

func mangakakalot() *scraper.Configuration {
	m := Mangabox(MangakakalotInfo.ID, MangakakalotInfo.Website, "/search/story/%s", "Jan 02,06", "span.chapter-time")

	// Most of the time redirects to chapmanganato.to,
	// no way to do checks or provide alternate urls on failed requests
	//
	// So far, looks like ids starting with "manga" are from manganato while
	// chapters starting with "read" are from mangakakalot
	m.GenerateSearchByIDURL = func(baseUrl, id string) (string, error) {
		// Only handle redirects to manganato itself
		if strings.HasPrefix(id, "manga") {
			return fmt.Sprintf("%s%s", "https://chapmanganato.to/", id), nil
		}
		return fmt.Sprintf("%s%s", baseUrl, id), nil
	}

	m.MangaExtractor.Selector += ", .panel_story_list .story_item, div.list-truyen-item-wrap"
	m.MangaExtractor.Title = func(selection *goquery.Selection) string {
		return selection.Find(".story_name a").Text()
	}
	m.MangaExtractor.URL = func(selection *goquery.Selection) string {
		return selection.Find(".story_name a").AttrOr("href", "")
	}

	return m
}
