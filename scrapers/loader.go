package scraper

import (
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/luevano/mangoprovider/mango/scraper"
	"github.com/luevano/mangoprovider/scrapers/asurascans"
	"github.com/luevano/mangoprovider/scrapers/flamescans"
	"github.com/luevano/mangoprovider/scrapers/manganato"
	"github.com/luevano/mangoprovider/scrapers/manganelo"
	"github.com/luevano/mangoprovider/scrapers/mangapill"
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	loaders := []libmangal.ProviderLoader{
		Loader(mangapill.ProviderInfo, mangapill.Options, options),
		Loader(asurascans.ProviderInfo, asurascans.Options, options),
		Loader(flamescans.ProviderInfo, flamescans.Options, options),
		Loader(manganato.ProviderInfo, manganato.Options, options),
		Loader(manganelo.ProviderInfo, manganelo.Options, options),
	}

	return loaders
}

func Loader(providerInfo libmangal.ProviderInfo, scraperOptions *scraper.Options, options mango.Options) libmangal.ProviderLoader {
	s, err := scraper.NewScraper(scraperOptions, options.HeadlessOptions)
	if err != nil {
		panic(err)
	}

	return mango.ProviderLoader{
		ProviderInfo: providerInfo,
		Options:      options,
		Funcs: mango.ProviderFuncs{
			SearchMangas:   s.SearchMangas,
			MangaVolumes:   s.MangaVolumes,
			VolumeChapters: s.VolumeChapters,
			ChapterPages:   s.ChapterPages,
		},
	}
}
