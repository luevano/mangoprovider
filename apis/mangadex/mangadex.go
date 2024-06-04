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
	Version:     "0.7.1",
	Description: "MangaDex scraper using mangodex",
	Website:     website,
}

type dex struct {
	client *mangodex.DexClient
	filter mango.Filter
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	d := dex{
		client: mangodex.NewDexClient(),
		filter: options.Filter,
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
