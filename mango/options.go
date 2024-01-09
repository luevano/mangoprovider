package mango

import (
	"net/http"

	"github.com/philippgille/gokv"
)

const (
	BundleID  = "mango"
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"
)

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
