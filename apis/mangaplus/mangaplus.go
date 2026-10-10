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
	Version:     "0.5.1",
	Description: "MangaPlus scraper using mangoplus",
	Website:     website,
}

type plus struct {
	client  *mangoplus.PlusClient
	filter  mango.FilterOptions
	options mango.MangaPlusOptions
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// Not using options.UserAgent, the API only accepts the one from the Android app
	o := options.MangaPlus
	p := plus{
		filter:  options.Filter,
		options: o,
	}

	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func() (mango.Functions, error) {
			client, err := mangoplus.NewPlusClient(o.Options)
			if err != nil {
				return mango.Functions{}, err
			}
			p.client = client

			return mango.Functions{
				SearchMangas:   p.SearchMangas,
				MangaVolumes:   p.MangaVolumes,
				VolumeChapters: p.VolumeChapters,
				ChapterPages:   p.ChapterPages,
				GetPageImage:   p.GetPageImage,
			}, nil
		},
	}
}
