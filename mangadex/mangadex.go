package mangadex

import (
	"context"
	"fmt"

	"github.com/bob620/mangodex"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
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
		Options: options,
		ProviderFuncs: mango.ProviderFuncs{
			FnSearchMangas: func(ctx context.Context, s string) ([]libmangal.Manga, error) {
				return nil, fmt.Errorf("unimplemented")
			},
			FnMangaVolumes: func(ctx context.Context, m libmangal.Manga) ([]libmangal.Manga, error) {
				return nil, fmt.Errorf("unimplemented")
			},
			FnVolumeChapters: func(ctx context.Context, v libmangal.Volume) ([]libmangal.Chapter, error) {
				return nil, fmt.Errorf("unimplemented")
			},
			FnChapterPages: func(ctx context.Context, c libmangal.Chapter) ([]libmangal.Page, error) {
				return nil, fmt.Errorf("unimplemented")
			},
			FnGetPageImage: func(ctx context.Context, p libmangal.Page) ([]byte, error) {
				return nil, fmt.Errorf("unimplemented")
			},
		},
	}
}
