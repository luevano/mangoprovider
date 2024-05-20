package mangathemesia

import (
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
)

var AsurascansInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-asurascans",
	Name:        "AsuraScans",
	Version:     "0.2.0",
	Description: "AsuraScans scraper",
	Website:     "https://asuracomic.net/",
}

var AsurascansConfig = Mangathemesia(AsurascansInfo.ID, AsurascansInfo.Website)
