package mangadex

import (
	"context"
	"net/http"
	"sync"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/fetch"
)

const website = "https://mangadex.org/"

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangadex",
	Name:        "MangaDex",
	Version:     "0.8.9",
	Description: "MangaDex scraper using mangodex",
	Website:     website,
	Settings:    settings,
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
			o := o
			if err := applySettings(options.Settings, &o); err != nil {
				return mango.Functions{}, err
			}
			dexOptions := o.Options
			if dexOptions.HTTPClient == nil {
				dexOptions.HTTPClient = options.HTTPClient
			}
			// 5xx only: a 429 goes up with its wait, hitting it again gets the IP banned
			// (api.mangadex.org/docs/2-limitations)
			// the MangaDex@Home reports are best-effort: a report endpoint that's down isn't
			// retried (and logged) for every page
			retry := fetch.Retry{Statuses: fetch.ServerErrors, Skip: func(req *http.Request) bool {
				return req.URL.String() == mangodex.MDHomeReportURL
			}}
			dexOptions.HTTPClient = fetch.RetryClient(dexOptions.HTTPClient, retry, nil, log)
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
