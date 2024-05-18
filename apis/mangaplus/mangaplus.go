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
	Version:     "0.1.0",
	Description: "MangaPlus scraper using mangoplus",
	Website:     website,
}

type plus struct {
	client *mangoplus.PlusClient
	filter mango.Filter
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	d := plus{
		client: mangoplus.NewPlusClient(),
		filter: options.Filter,
	}

	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func() mango.Functions {
			return mango.Functions{
				SearchMangas:   d.SearchMangas,
				MangaVolumes:   d.MangaVolumes,
				VolumeChapters: d.VolumeChapters,
				ChapterPages:   d.ChapterPages,
				GetPageImage:   d.GetPageImage,
			}
		},
	}
}
