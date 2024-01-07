package mangadex

import (
	"context"
	"fmt"

	"github.com/bob620/mangodex"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

var dex = mangodex.NewDexClient()

var dexProviderInfo = libmangal.ProviderInfo{
	ID:          "mangadex",
	Name:        "Mangadex",
	Version:     "0.1.0",
	Description: "Mangadex scraper using mangodex",
	Website:     "https://mangadex.org/",
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	if err := dexProviderInfo.Validate(); err != nil {
		return nil
	}

	return mango.MangoLoader{
		ProviderInfo: dexProviderInfo,
		Options:      options,
		Funcs: mango.ProviderFuncs{
			SearchMangas: func(ctx context.Context, store gokv.Store, s string) ([]libmangal.Manga, error) {
				return nil, fmt.Errorf("unimplemented")
			},
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
