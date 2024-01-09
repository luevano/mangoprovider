package mango

import (
	"context"

	"github.com/luevano/libmangal"
)

var _ libmangal.ProviderLoader = (*ProviderLoader)(nil)

type ProviderLoader struct {
	libmangal.ProviderInfo
	Options Options
	Funcs   ProviderFuncs
}

func (l ProviderLoader) String() string {
	return l.Name
}

func (l ProviderLoader) Info() libmangal.ProviderInfo {
	return l.ProviderInfo
}

func (l ProviderLoader) Load(ctx context.Context) (libmangal.Provider, error) {
	provider := &Provider{
		ProviderInfo: l.ProviderInfo,
		Options:      l.Options,
		Funcs:        l.Funcs,
	}

	store, err := l.Options.HTTPStoreProvider(l.ProviderInfo.ID)
	if err != nil {
		return nil, err
	}

	provider.store = store

	return provider, nil
}
