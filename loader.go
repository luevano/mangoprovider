package mangoprovider

import (
	"context"
	"errors"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

var _ libmangal.ProviderLoader = (*Loader)(nil)

type Loader struct {
	libmangal.ProviderInfo
	Options Options
	F       func(ctx context.Context, log *Logger) (Functions, error) // So that the providers are only built on ProviderLoader.Load(ctx)
	// Account: The provider has a user account (Functions.Account), known before loading it.
	Account bool
}

// HasAccount: The provider has a user account, see libmangal.AccountProvider.
func (l *Loader) HasAccount() bool {
	return l.Account
}

func (l *Loader) String() string {
	return l.Name
}

func (l *Loader) Info() libmangal.ProviderInfo {
	return l.ProviderInfo
}

func (l *Loader) Load(ctx context.Context) (libmangal.Provider, error) {
	log := NewProviderLogger(l.ProviderInfo)
	store, err := openStore(func(bucketName string) (gokv.Store, error) {
		return l.Options.CacheStore(l.ProviderInfo.ID, bucketName)
	}, log)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", l.ProviderInfo.Name, err)
	}

	f, err := l.F(ctx, log)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("load %s: %w", l.ProviderInfo.Name, shared(err)), store.Close())
	}

	return &Provider{
		ProviderInfo: l.ProviderInfo,
		Options:      l.Options,
		F:            f,
		client:       l.Options.HTTPClient,
		store:        store,
		log:          log,
	}, nil
}
