package scrapers

import (
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
	"github.com/luevano/mangoprovider/scrapers/asurascans"
	"github.com/luevano/mangoprovider/scrapers/flamescans"
	"github.com/luevano/mangoprovider/scrapers/mangabox"
	"github.com/luevano/mangoprovider/scrapers/mangapill"
	"github.com/luevano/mangoprovider/scrapers/mangaplus"
	"github.com/luevano/mangoprovider/scrapers/mangasee"
	"github.com/luevano/mangoprovider/scrapers/toonily"
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	loaders := []libmangal.ProviderLoader{
		Loader(mangaplus.Info, mangaplus.Config, options),
		Loader(mangapill.Info, mangapill.Config, options),
		Loader(mangasee.Info, mangasee.Config, options),
		Loader(asurascans.Info, asurascans.Config, options),
		Loader(flamescans.Info, flamescans.Config, options),
		// Mangabox
		Loader(mangabox.ManganatoInfo, mangabox.ManganatoConfig, options),
		Loader(mangabox.MangabatInfo, mangabox.MangabatConfig, options),
		Loader(mangabox.MangairoInfo, mangabox.MangairoConfig, options),
		Loader(mangabox.MangakakalotInfo, mangabox.MangakakalotConfig, options),

		Loader(toonily.Info, toonily.Config, options),
	}

	return loaders
}

func Loader(info libmangal.ProviderInfo, config *scraper.Configuration, options mango.Options) libmangal.ProviderLoader {
	return &mango.Loader{
		ProviderInfo: info,
		Options:      options,
		F: func() mango.Functions {
			s, err := scraper.NewScraper(config, options)
			if err != nil {
				panic(err)
			}
			return mango.Functions{
				SearchMangas:   s.SearchMangas,
				MangaVolumes:   s.MangaVolumes,
				VolumeChapters: s.VolumeChapters,
				ChapterPages:   s.ChapterPages,
			}
		},
	}
}
