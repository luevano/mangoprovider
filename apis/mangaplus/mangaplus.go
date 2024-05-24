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
	Version:     "0.4.4",
	Description: "MangaPlus scraper using mangoplus",
	Website:     website,
}

type plus struct {
	client    *mangoplus.PlusClient
	userAgent string
	filter    mango.Filter
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// TODO: decide if this should be moved into mangal,
	// kind of a mess of "options" moving around
	o := mangoplus.DefaultOptions()
	o.UserAgent = options.UserAgent

	mPO := options.MangaPlus
	if mPO.OSVersion != "" {
		o.OSVersion = mPO.OSVersion
	}
	if mPO.AppVersion != "" {
		o.AppVersion = mPO.AppVersion
	}
	if mPO.AndroidID != "" {
		o.AndroidID = mPO.AndroidID
	}

	plusClient := mangoplus.NewPlusClient(o)
	p := plus{
		client:    plusClient,
		userAgent: options.UserAgent,
		filter:    options.Filter,
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
