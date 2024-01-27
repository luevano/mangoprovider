package mangoprovider

import (
	"context"

	"github.com/luevano/libmangal"
)

var _ libmangal.ProviderLoader = (*Loader)(nil)

type Loader struct {
	libmangal.ProviderInfo
	Options Options
	F       func() Functions // So that the scrapers are loaded on ProviderLoader.Load(ctx)
}

func (l Loader) String() string {
	return l.Name
}

func (l Loader) Info() libmangal.ProviderInfo {
	return l.ProviderInfo
}

func (l Loader) Load(ctx context.Context) (libmangal.Provider, error) {
	store, err := l.Options.HTTPStore(l.ProviderInfo.ID)
	if err != nil {
		return nil, err
	}

	return &Provider{
		ProviderInfo: l.ProviderInfo,
		Options:      l.Options,
		F:            l.F(),
		store:        store,
	}, nil
}
