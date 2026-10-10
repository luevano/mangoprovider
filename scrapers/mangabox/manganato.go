package mangabox

import (
	"fmt"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var ManganatoInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-manganato",
	Name:        "Manganato",
	Version:     "0.3.1",
	Description: "Manganato scraper",
	Website:     "https://manganato.com/",
}

// ManganatoConfig: A new configuration, built on every load.
func ManganatoConfig(_ *mango.Logger) *scraper.Configuration {
	m := Mangabox(ManganatoInfo, "/search/story/%s", "Jan 02,06", "span.chapter-time")

	m.GenerateSearchByIDURL = func(_, id string) (string, error) {
		return fmt.Sprintf("%s%s", "https://chapmanganato.to/", id), nil
	}

	return m
}
