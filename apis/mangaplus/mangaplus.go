package mangaplus

import (
	"context"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/fetch"
)

const website = "https://mangaplus.shueisha.co.jp/"

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-mangaplus",
	Name:        "MangaPlus",
	Version:     "0.6.9",
	Description: "MangaPlus scraper using mangoplus",
	Website:     website,
	Settings:    settings,
	Traits: libmangal.ProviderTraits{
		// MangaPlus shows only some of a series' chapters (often the first and latest)
		PartialChapters: true,
		// the device gets locked out after bursts of requests (mangoplus.ErrRateLimited)
		AvoidUnattended: true,
	},
}

type plus struct {
	client  *mangoplus.PlusClient
	filter  mango.FilterOptions
	options mango.MangaPlusOptions
	log     *mango.Logger
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	// Not using options.UserAgent, the API only accepts the one from the Android app
	o := options.MangaPlus

	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		Account:      true,
		F: func(_ context.Context, log *mango.Logger) (mango.Functions, error) {
			o := o
			if err := applySettings(options.Settings, &o, log); err != nil {
				return mango.Functions{}, err
			}
			plusOptions := o.Options
			// the website's api takes a browser's
			if plusOptions.WebUserAgent == "" {
				plusOptions.WebUserAgent = options.UserAgent
			}
			if plusOptions.HTTPClient == nil {
				plusOptions.HTTPClient = options.HTTPClient
			}
			// 5xx only: retrying a 429 within seconds can't beat the lockout (see mangoplus.ErrRateLimited)
			plusOptions.HTTPClient = fetch.RetryClient(plusOptions.HTTPClient, fetch.Retry{Statuses: fetch.ServerErrors}, nil, log)
			client, err := mangoplus.NewPlusClient(plusOptions)
			if err != nil {
				return mango.Functions{}, err
			}
			p := &plus{
				client:  client,
				filter:  options.Filter,
				options: o,
				log:     log,
			}

			return mango.Functions{
				SearchMangas:   p.SearchMangas,
				MangaByID:      p.MangaByID,
				MangaVolumes:   p.MangaVolumes,
				VolumeChapters: p.VolumeChapters,
				ChapterPages:   p.ChapterPages,
				GetPageImage:   p.GetPageImage,
				Account:        p.Account,
			}, nil
		},
	}
}

// cacheKey: key, apart for the web api (it lists fewer chapters).
func (p *plus) cacheKey(key string) string {
	if p.client != nil && p.client.WebAPI() {
		return key + "-web"
	}
	return key
}
