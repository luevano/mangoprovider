package mangabox

import (
	"fmt"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var MangabatInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangabat",
	Name:        "Mangabat",
	Version:     "0.2.0",
	Description: "Mangabat scraper",
	Website:     "https://h.mangabat.com/",
}

var MangabatConfig = mangabat()

func mangabat() *scraper.Configuration {
	m := Mangabox(MangabatInfo.ID, MangabatInfo.Website, "/search/manga/%s", "Jan 02,06", "span.chapter-time")

	m.GenerateSearchByIDURL = func(_, id string) (string, error) {
		return fmt.Sprintf("%s%s", "https://readmangabat.com/", id), nil
	}
	m.MangaExtractor.Selector = ".panel-list-story .list-story-item"

	return m
}
