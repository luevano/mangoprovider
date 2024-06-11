package mangapluscreators

import (
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus/creators"
	mango "github.com/luevano/mangoprovider"
)

const website = "https://medibang.com/mpc/"

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangapluscreators",
	Name:        "MangaPlusCreators",
	Version:     "0.1.1",
	Description: "MangaPlusCreators (MPC) scraper using mangoplus",
	Website:     website,
}

type mpc struct {
	client  *creators.CreatorsClient
	filter  mango.FilterOptions
	options mango.MangaPlusCreatorsOptions
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// Update the user agent with the actual one from the options
	o := options.MangaPlusCreators
	o.UserAgent = options.UserAgent
	c := mpc{
		client:  creators.NewCreatorsClient(o.Options),
		filter:  options.Filter,
		options: o,
	}

	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func() mango.Functions {
			return mango.Functions{
				SearchMangas:   c.SearchMangas,
				MangaVolumes:   c.MangaVolumes,
				VolumeChapters: c.VolumeChapters,
				ChapterPages:   c.ChapterPages,
			}
		},
	}
}
