package scrapers

import (
	"context"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
	"github.com/luevano/mangoprovider/scrapers/mangapill"
	"github.com/luevano/mangoprovider/scrapers/toonily"
	"github.com/luevano/mangoprovider/scrapers/weebcentral"
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	loaders := []libmangal.ProviderLoader{
		Loader(mangapill.Info, mangapill.Config, options),
		Loader(weebcentral.Info, weebcentral.Config, options),

		Loader(toonily.Info, toonily.Config, options),
	}

	return loaders
}

func Loader(info libmangal.ProviderInfo, newConfig func(*mango.Logger) *scraper.Configuration, options mango.Options) libmangal.ProviderLoader {
	return &mango.Loader{
		ProviderInfo: info,
		Options:      options,
		F: func(ctx context.Context, log *mango.Logger) (mango.Functions, error) {
			s, err := scraper.NewScraper(ctx, newConfig(log), options, log)
			if err != nil {
				return mango.Functions{}, err
			}
			return mango.Functions{
				SearchMangas:   s.SearchMangas,
				MangaVolumes:   s.MangaVolumes,
				VolumeChapters: s.VolumeChapters,
				ChapterPages:   s.ChapterPages,
				Close:          s.Close,
			}, nil
		},
	}
}
