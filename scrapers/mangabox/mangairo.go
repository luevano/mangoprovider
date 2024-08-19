package mangabox

import (
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var MangairoInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangairo",
	Name:        "Mangairo",
	Version:     "0.2.1",
	Description: "Mangairo scraper",
	Website:     "https://w.mangairo.com/",
}

var MangairoConfig = mangairo()

func mangairo() *scraper.Configuration {
	m := Mangabox(MangairoInfo.ID, MangairoInfo.Website, "/list/search/%s", "Jan-02-06", "p")

	m.GenerateSearchByIDURL = func(_, id string) (string, error) {
		return fmt.Sprintf("%s%s", "https://chap.mangairo.com/", id), nil
	}

	m.MangaByIDExtractor.Selector += ", div.story_content"
	m.MangaByIDExtractor.Cover = func(selection *goquery.Selection) string {
		return selection.Find("div.manga-info-pic img, span.info-image img, div.story_info_left img").AttrOr("src", "")
	}

	m.MangaExtractor.Selector = ".story-list .story-item"
	m.MangaExtractor.Title = func(selection *goquery.Selection) string {
		return selection.Find(".story-name a").Text()
	}
	m.MangaExtractor.URL = func(selection *goquery.Selection) string {
		return selection.Find(".story-name a").AttrOr("href", "")
	}

	m.ChapterExtractor.Selector = fmt.Sprintf("%s, div#chapter_list li", m.ChapterExtractor.Selector)
	m.PageExtractor.Selector = fmt.Sprintf("%s, div.panel-read-story img", m.PageExtractor.Selector)

	return m
}
