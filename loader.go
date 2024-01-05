package mangoprovider

import (
	"context"
	"net/http"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
)

var _ libmangal.ProviderLoader = (*mangoLoader)(nil)

type Options struct {
	HTTPClient        *http.Client
	HTTPStoreProvider func() (gokv.Store, error)
}

func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{},
		HTTPStoreProvider: func() (gokv.Store, error) {
			return syncmap.NewStore(syncmap.DefaultOptions), nil
		},
	}
}

// NewLoader is the entry point for external calls
func NewLoader(info libmangal.ProviderInfo, options Options) (libmangal.ProviderLoader, error) {
	if err := info.Validate(); err != nil {
		return nil, err
	}

	return mangoLoader{
		ProviderInfo: info,
		options:      options,
	}, nil
}

type mangoLoader struct {
	libmangal.ProviderInfo
	options Options
}

func (l mangoLoader) String() string {
	return l.Name
}

func (l mangoLoader) Info() libmangal.ProviderInfo {
	return l.ProviderInfo
}

func (l mangoLoader) Load(ctx context.Context) (libmangal.Provider, error) {
	provider := &mangoProvider{
		ProviderInfo: l.ProviderInfo,
		options:      l.options,
	}

	store, err := l.options.HTTPStoreProvider()
	if err != nil {
		return nil, err
	}

	provider.store = store

	return provider, nil
}
