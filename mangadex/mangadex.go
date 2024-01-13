package mangadex

import (
	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	"github.com/luevano/mangoprovider/mango"
)

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangadex",
	Name:        "Mangadex",
	Version:     "0.3.1",
	Description: "Mangadex scraper using mangodex",
	Website:     "https://mangadex.org/",
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

	// TODO: use mangodex get chapter page for downloading, instead of the mangoloader generic one
	return mango.ProviderLoader{
		ProviderInfo: providerInfo,
		Options:      options,
		Funcs: mango.ProviderFuncs{
			SearchMangas:   d.SearchMangas,
			MangaVolumes:   d.MangaVolumes,
			VolumeChapters: d.VolumeChapters,
			ChapterPages:   d.ChapterPages,
		},
	}
}
