package mangabox

import (
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var MangakakalotInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangakakalot",
	Name:        "Mangakakalot",
	Version:     "0.1.0",
	Description: "Mangakakalot scraper",
	Website:     "https://mangakakalot.com/",
}

var MangakakalotConfig = mangakakalot()

func mangakakalot() *scraper.Configuration {
	m := Mangabox(MangakakalotInfo.ID, MangakakalotInfo.Website, "/search/story/%s", "Jan 02,06", "span.chapter-time")
	m.MangaExtractor.Selector = fmt.Sprintf("%s, .panel_story_list .story_item, div.list-truyen-item-wrap", m.MangaExtractor.Selector)
	m.MangaExtractor.Title = func(selection *goquery.Selection) string {
		return selection.Find(".story_name a").Text()
	}
	m.MangaExtractor.URL = func(selection *goquery.Selection) string {
		return selection.Find(".story_name a").AttrOr("href", "")
	}

	return m
}
