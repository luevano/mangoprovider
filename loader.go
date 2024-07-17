package mangoprovider

import (
	"context"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

var _ libmangal.ProviderLoader = (*Loader)(nil)

type Loader struct {
	libmangal.ProviderInfo
	Options Options
	F       func() Functions // So that the scrapers are loaded on ProviderLoader.Load(ctx)
}

func (l *Loader) String() string {
	return l.Name
}

func (l *Loader) Info() libmangal.ProviderInfo {
	return l.ProviderInfo
}

func (l *Loader) Load(ctx context.Context) (libmangal.Provider, error) {
	// gokv.Store wrapper
	store := Store{
		openStore: func(bucketName string) (gokv.Store, error) {
			return l.Options.CacheStore(l.ProviderInfo.ID, bucketName)
		},
	}

	return &Provider{
		ProviderInfo: l.ProviderInfo,
		Options:      l.Options,
		F:            l.F(),
		client:       l.Options.HTTPClient,
		store:        store,
	}, nil
}
