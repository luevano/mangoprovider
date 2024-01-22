package scrapers

import (
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
	"github.com/luevano/mangoprovider/scraper/headless"
	"github.com/luevano/mangoprovider/scrapers/asurascans"
	"github.com/luevano/mangoprovider/scrapers/flamescans"
	"github.com/luevano/mangoprovider/scrapers/manganato"
	"github.com/luevano/mangoprovider/scrapers/manganelo"
	"github.com/luevano/mangoprovider/scrapers/mangapill"
	"github.com/luevano/mangoprovider/scrapers/toonily"
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	loaders := []libmangal.ProviderLoader{
		Loader(mangapill.ProviderInfo, mangapill.Options, options),
		Loader(asurascans.ProviderInfo, asurascans.Options, options),
		Loader(flamescans.ProviderInfo, flamescans.Options, options),
		Loader(manganato.ProviderInfo, manganato.Options, options),
		Loader(manganelo.ProviderInfo, manganelo.Options, options),
		Loader(toonily.ProviderInfo, toonily.Options, options),
	}

	return loaders
}

func Loader(providerInfo libmangal.ProviderInfo, scraperOptions *scraper.Options, options mango.Options) libmangal.ProviderLoader {
	// Could also ve directly converted as they structs are identical?
	// s, err := scraper.NewScraper(scraperOptions, headless.Options(options.Headless))
	headlessOptions := headless.Options{
		UseFlaresolverr: options.Headless.UseFlaresolverr,
		FlaresolverrURL: options.Headless.FlaresolverrURL,
	}
	s, err := scraper.NewScraper(scraperOptions, headlessOptions)
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
