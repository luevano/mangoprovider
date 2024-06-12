package mangaplus

import (
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
)

const website = "https://mangaplus.shueisha.co.jp/"

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangaplus",
	Name:        "MangaPlus",
	Version:     "0.5.0",
	Description: "MangaPlus scraper using mangoplus",
	Website:     website,
}

type plus struct {
	client  *mangoplus.PlusClient
	filter  mango.FilterOptions
	options mango.MangaPlusOptions
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// Update the user agent with the actual one from the options
	o := options.MangaPlus
	o.UserAgent = options.UserAgent
	p := plus{
		client:  mangoplus.NewPlusClient(o.Options),
		filter:  options.Filter,
		options: o,
	}

	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func() mango.Functions {
			return mango.Functions{
				SearchMangas:   p.SearchMangas,
				MangaVolumes:   p.MangaVolumes,
				VolumeChapters: p.VolumeChapters,
				ChapterPages:   p.ChapterPages,
				GetPageImage:   p.GetPageImage,
			}
		},
	}
}
