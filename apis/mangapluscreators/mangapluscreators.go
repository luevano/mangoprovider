package mangapluscreators

import (
	"context"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus/creators"
	mango "github.com/luevano/mangoprovider"
)

const website = "https://medibang.com/mpc/"

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangapluscreators",
	Name:        "MangaPlusCreators",
	Version:     "0.2.2",
	Description: "MangaPlusCreators (MPC) scraper using mangoplus",
	Website:     website,
}

type mpc struct {
	client  *creators.CreatorsClient
	filter  mango.FilterOptions
	options mango.MangaPlusCreatorsOptions
	log     *mango.Logger
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// Update the user agent with the actual one from the options
	o := options.MangaPlusCreators
	o.UserAgent = options.UserAgent
	c := mpc{
		filter:  options.Filter,
		options: o,
	}

	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func(_ context.Context, log *mango.Logger) (mango.Functions, error) {
			c.log = log
			mpcOptions := o.Options
			if mpcOptions.HTTPClient == nil {
				mpcOptions.HTTPClient = options.HTTPClient
			}
			client, err := creators.NewCreatorsClient(mpcOptions)
			if err != nil {
				return mango.Functions{}, err
			}
			c.client = client

			return mango.Functions{
				SearchMangas:   c.SearchMangas,
				MangaVolumes:   c.MangaVolumes,
				VolumeChapters: c.VolumeChapters,
				ChapterPages:   c.ChapterPages,
			}, nil
		},
	}
}
