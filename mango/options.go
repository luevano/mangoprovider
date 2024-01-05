package mango

import (
	"net/http"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
)

type DexOptions struct{}

type Options struct {
	HTTPClient        *http.Client
	HTTPStoreProvider func() (gokv.Store, error)
	MangadexOptions   DexOptions
}

func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{},
		HTTPStoreProvider: func() (gokv.Store, error) {
			return syncmap.NewStore(syncmap.DefaultOptions), nil
		},
	}
}
