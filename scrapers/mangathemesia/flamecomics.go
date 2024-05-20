package mangathemesia

import (
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
)

var FlamecomicsInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-flamecomics",
	Name:        "FlameComics",
	Version:     "0.2.0",
	Description: "FlameComics scraper",
	Website:     "https://flamecomics.com/",
}

// FlameComics in tachiyomi (RIP) has a really complicated
// logic for dealing with "composite images" whatever that means, gotta keep an eye
var FlamecomicsConfig = Mangathemesia(FlamecomicsInfo.ID, FlamecomicsInfo.Website)
