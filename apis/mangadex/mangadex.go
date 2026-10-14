package mangadex

import (
	"context"
	"sync"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

const website = "https://mangadex.org/"

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangadex",
	Name:        "MangaDex",
	Version:     "0.8.3",
	Description: "MangaDex scraper using mangodex",
	Website:     website,
}

type dex struct {
	client  *mangodex.DexClient
	filter  mango.FilterOptions
	options mango.MangaDexOptions
	log     *mango.Logger

	// atHomeMu guards the chapters' AtHome, shared by their pages
	atHomeMu sync.Mutex
	// atHomeRefreshed: the last MangaDex@Home server fetched again for a chapter, by its id
	atHomeRefreshed map[string]*mangodex.AtHomeServer
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// Not using options.UserAgent, MangaDex doesn't allow spoofed (browser) user agents
	o := options.MangaDex

	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func(_ context.Context, log *mango.Logger) (mango.Functions, error) {
			dexOptions := o.Options
			if dexOptions.HTTPClient == nil {
				dexOptions.HTTPClient = options.HTTPClient
			}
			client, err := mangodex.NewDexClient(dexOptions)
			if err != nil {
				return mango.Functions{}, err
			}
			d := &dex{
				client:  client,
				filter:  options.Filter,
				options: o,
				log:     log,

				atHomeRefreshed: map[string]*mangodex.AtHomeServer{},
			}

			return mango.Functions{
				SearchMangas:   d.SearchMangas,
				MangaByID:      d.MangaByID,
				MangaVolumes:   d.MangaVolumes,
				VolumeChapters: d.VolumeChapters,
				ChapterPages:   d.ChapterPages,
				GetPageImage:   d.GetPageImage,
			}, nil
		},
	}
}
