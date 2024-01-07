package mango

import (
	"context"

	"github.com/luevano/libmangal"
)

var _ libmangal.ProviderLoader = (*MangoLoader)(nil)

type MangoLoader struct {
	libmangal.ProviderInfo
	Options Options
	Funcs   ProviderFuncs
}

func (l MangoLoader) String() string {
	return l.Name
}

func (l MangoLoader) Info() libmangal.ProviderInfo {
	return l.ProviderInfo
}

func (l MangoLoader) Load(ctx context.Context) (libmangal.Provider, error) {
	provider := &MangoProvider{
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
