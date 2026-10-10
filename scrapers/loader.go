package scrapers

import (
	"context"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
	"github.com/luevano/mangoprovider/scrapers/batoto"
	"github.com/luevano/mangoprovider/scrapers/mangabox"
	"github.com/luevano/mangoprovider/scrapers/mangapill"
	"github.com/luevano/mangoprovider/scrapers/mangasee"
	"github.com/luevano/mangoprovider/scrapers/mangathemesia"
	"github.com/luevano/mangoprovider/scrapers/toonily"
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	loaders := []libmangal.ProviderLoader{
		Loader(mangapill.Info, mangapill.Config, options),
		Loader(mangasee.Info, mangasee.Config, options),
		Loader(batoto.Info, batoto.Config, options),
		// Mangathemesia
		Loader(mangathemesia.AsurascansInfo, mangathemesia.AsurascansConfig, options),
		Loader(mangathemesia.FlamecomicsInfo, mangathemesia.FlamecomicsConfig, options),
		// Mangabox
		Loader(mangabox.ManganatoInfo, mangabox.ManganatoConfig, options),
		Loader(mangabox.MangabatInfo, mangabox.MangabatConfig, options),
		Loader(mangabox.MangairoInfo, mangabox.MangairoConfig, options),
		Loader(mangabox.MangakakalotInfo, mangabox.MangakakalotConfig, options),

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
