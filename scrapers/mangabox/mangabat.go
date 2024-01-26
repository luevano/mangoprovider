package mangabox

import (
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var MangabatInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangabat",
	Name:        "Mangabat",
	Version:     "0.1.0",
	Description: "Mangabat scraper",
	Website:     "https://h.mangabat.com/",
}

var MangabatConfig = mangabat()

func mangabat() *scraper.Configuration {
	m := Mangabox(MangabatInfo.ID, MangabatInfo.Website, "/search/manga/%s", "Jan 02,06")

	m.MangaExtractor.Selector = "div.list-story-item"

	return m
}
