package mangabox

import (
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var ManganatoInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-manganato",
	Name:        "Manganato",
	Version:     "0.2.0",
	Description: "Manganato scraper",
	Website:     "https://manganato.com/",
}

var ManganatoConfig = manganato()

func manganato() *scraper.Configuration {
	return Mangabox(ManganatoInfo.ID, ManganatoInfo.Website, "/search/story/%s", "Jan 02,06", "span.chapter-time")
}
