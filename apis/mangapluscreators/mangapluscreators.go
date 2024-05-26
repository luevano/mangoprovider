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
	Version:     "0.1.0",
	Description: "MangaPlusCreators (MPC) scraper using mangoplus",
	Website:     website,
}

type mpc struct {
	client    *creators.CreatorsClient
	userAgent string
	filter    mango.Filter
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// TODO: decide if this should be moved into mangal,
	// kind of a mess of "options" moving around
	o := creators.DefaultOptions()
	o.UserAgent = options.UserAgent

	plusClient := creators.NewCreatorsClient(o)
	c := mpc{
		client:    plusClient,
		userAgent: options.UserAgent,
		filter:    options.Filter,
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
