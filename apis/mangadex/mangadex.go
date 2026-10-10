package mangadex

import (
	"context"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

const website = "https://mangadex.org/"

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangadex",
	Name:        "MangaDex",
	Version:     "0.7.2",
	Description: "MangaDex scraper using mangodex",
	Website:     website,
}

type dex struct {
	client  *mangodex.DexClient
	filter  mango.FilterOptions
	options mango.MangaDexOptions
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// Not using options.UserAgent, MangaDex doesn't allow spoofed (browser) user agents
	o := options.MangaDex
	d := dex{
		filter:  options.Filter,
		options: o,
	}

	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func(_ context.Context) (mango.Functions, error) {
			client, err := mangodex.NewDexClient(o.Options)
			if err != nil {
				return mango.Functions{}, err
			}
			d.client = client

			return mango.Functions{
				SearchMangas:   d.SearchMangas,
				MangaVolumes:   d.MangaVolumes,
				VolumeChapters: d.VolumeChapters,
				ChapterPages:   d.ChapterPages,
				GetPageImage:   d.GetPageImage,
			}, nil
		},
	}
}
