package mangadex

import (
	"context"
	"fmt"

	"github.com/luevano/mangodex"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
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

	return mango.MangoLoader{
		ProviderInfo: DexProviderInfo,
		Options:      options,
		Funcs: mango.ProviderFuncs{
			SearchMangas: dex.SearchMangas,
			MangaVolumes: func(ctx context.Context, store gokv.Store, m mango.MangoManga) ([]libmangal.Volume, error) {
				return nil, fmt.Errorf("unimplemented")
			},
			VolumeChapters: func(ctx context.Context, store gokv.Store, v mango.MangoVolume) ([]libmangal.Chapter, error) {
				return nil, fmt.Errorf("unimplemented")
			},
			ChapterPages: func(ctx context.Context, store gokv.Store, c mango.MangoChapter) ([]libmangal.Page, error) {
				return nil, fmt.Errorf("unimplemented")
			},
		},
	}
}
