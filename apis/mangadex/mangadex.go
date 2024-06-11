package mangadex

import (
	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

const website = "https://mangadex.org/"

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangadex",
	Name:        "MangaDex",
	Version:     "0.7.2",
	Description: "MangaDex scraper using mangodex",
	Website:     website,
}

type dex struct {
	client  *mangodex.DexClient
	filter  mango.FilterOptions
	options mango.MangaDexOptions
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// Update the user agent with the actual one from the options
	o := options.MangaDex
	o.UserAgent = options.UserAgent
	d := dex{
		client:  mangodex.NewDexClient(o.Options),
		filter:  options.Filter,
		options: o,
	}

	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func() mango.Functions {
			return mango.Functions{
				SearchMangas:   d.SearchMangas,
				MangaVolumes:   d.MangaVolumes,
				VolumeChapters: d.VolumeChapters,
				ChapterPages:   d.ChapterPages,
				GetPageImage:   d.GetPageImage,
			}
		},
	}
}
