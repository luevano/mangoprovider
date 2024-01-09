package mango

import (
	"net/http"

	"github.com/philippgille/gokv"
)

const BundleID = "mango"

type DexOptions struct {
	NSFW               bool
	Language           string
	TitleChapterNumber bool
}

type Options struct {
	HTTPClient        *http.Client
	HTTPStoreProvider func(providerID string) (gokv.Store, error)
	MangadexOptions   DexOptions
}
