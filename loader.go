package mangoprovider

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

var _ libmangal.ProviderLoader = (*Loader)(nil)

type Loader struct {
	libmangal.ProviderInfo
	Options Options
	F       func(ctx context.Context) (Functions, error) // So that the providers are only built on ProviderLoader.Load(ctx)
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

	f, err := l.F(ctx)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", l.ProviderInfo.Name, err)
	}

	return &Provider{
		ProviderInfo: l.ProviderInfo,
		Options:      l.Options,
		F:            f,
		client:       l.Options.HTTPClient,
		store:        store,
	}, nil
}
