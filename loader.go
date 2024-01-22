package mangoprovider

import (
	"context"

	"github.com/luevano/libmangal"
)

var _ libmangal.ProviderLoader = (*Loader)(nil)

type Loader struct {
	libmangal.ProviderInfo
	Options Options
	F   Functions
}

func (l Loader) String() string {
	return l.Name
}

func (l Loader) Info() libmangal.ProviderInfo {
	return l.ProviderInfo
}

func (l Loader) Load(ctx context.Context) (libmangal.Provider, error) {
	provider := &Provider{
		ProviderInfo: l.ProviderInfo,
		Options:      l.Options,
		F:        l.F,
	}

	store, err := l.Options.HTTPStoreProvider(l.ProviderInfo.ID)
	if err != nil {
		return nil, err
	}

	provider.store = store

	return provider, nil
}
