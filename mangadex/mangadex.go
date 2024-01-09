package mangadex

import (
	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	"github.com/luevano/mangoprovider/mango"
)

var DexProviderInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangadex",
	Name:        "Mangadex",
	Version:     "0.1.0",
	Description: "Mangadex scraper using mangodex",
	Website:     "https://mangadex.org/",
}

type Dex struct {
	client  *mangodex.DexClient
	options mango.DexOptions
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	dex := Dex{
		client:  mangodex.NewDexClient(),
		options: options.MangadexOptions,
	}

	// TODO: use mangodex get chapter page for downloading, instead of the mangoloader generic one
	return mango.ProviderLoader{
		ProviderInfo: DexProviderInfo,
		Options:      options,
		Funcs: mango.ProviderFuncs{
			SearchMangas:   dex.SearchMangas,
			MangaVolumes:   dex.MangaVolumes,
			VolumeChapters: dex.VolumeChapters,
			ChapterPages:   dex.ChapterPages,
		},
	}
}
