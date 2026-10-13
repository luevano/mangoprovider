package mangoprovider

import (
	"net/http"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	"github.com/luevano/mangoplus"
	"github.com/luevano/mangoplus/creators"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/encoding"
	"github.com/philippgille/gokv/syncmap"
)

const BundleID = "mango"

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36"

// FilterOptions are general filtering options for all sources.
type FilterOptions = libmangal.FilterOptions

func DefaultFilterOptions() FilterOptions {
	return FilterOptions{
		NSFW:                    false,
		Language:                "en",
		AvoidDuplicateChapters:  true,
		ShowUnavailableChapters: false,
	}
}

// HeadlessOptions are options that applied to the Headless browser,
// that some sources use for scraping.
type HeadlessOptions struct {
	// UseFlaresolverr if Flaresolverr should be used for the request.
	UseFlaresolverr bool

	// FlaresolverrURL the URL of the Flaresolverr API.
	FlaresolverrURL string
}

func DefaultHeadlessOptions() HeadlessOptions {
	return HeadlessOptions{
		UseFlaresolverr: false,
		FlaresolverrURL: "http://localhost:8191/v1",
	}
}

// MangaDexOptions are options that apply to the MangaDex source.
type MangaDexOptions struct {
	// Options from the mangodex API package.
	mangodex.Options

	// DataSaver if lower quality images should be requested.
	DataSaver bool
}

func DefaultMangaDexOptions() MangaDexOptions {
	o := mangodex.DefaultOptions()
	return MangaDexOptions{
		Options:   o,
		DataSaver: false,
	}
}

// MangaPlusOptions are options that apply to the MangaPlus source.
type MangaPlusOptions struct {
	// Options from the mangoplus API package.
	mangoplus.Options

	// Quality of the images that should be requested.
	Quality string
}

func DefaultMangaPlusOptions() MangaPlusOptions {
	o := mangoplus.DefaultOptions()
	return MangaPlusOptions{
		Options: o,
		Quality: "super_high",
	}
}

// MangaPlusCreatorsOptions are options that apply to the MangaPlusCreators source.
type MangaPlusCreatorsOptions struct {
	// Options from the mangoplus API package.
	creators.Options
}

func DefaultMangaPlusCreatorsOptions() MangaPlusCreatorsOptions {
	o := creators.DefaultOptions()
	return MangaPlusCreatorsOptions{
		Options: o,
	}
}

// Options are the general mangoprovider options.
type Options struct {
	// HTTPClient HTTP client to use for all requests.
	HTTPClient *http.Client

	// UserAgent to use for all HTTP requests.
	UserAgent string

	// CacheStore returns a gokv.Store implementation for use as a cache storage.
	CacheStore func(dbName, bucketName string) (gokv.Store, error)

	// Parallelism: How many html requests the scrapers make at once, at most 4.
	Parallelism uint8

	// Filter options.
	Filter FilterOptions

	// Headless options.
	Headless HeadlessOptions

	// Mirrors: the preferred website (its Website or one of its Mirrors) by provider id.
	Mirrors map[string]string

	// MangaDex options
	MangaDex MangaDexOptions

	// MangaPlus options.
	MangaPlus MangaPlusOptions

	// MangaPlusCreators options.
	MangaPlusCreators MangaPlusCreatorsOptions
}

func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{Timeout: time.Minute},
		UserAgent:  defaultUserAgent,
		CacheStore: func(dbName, bucketName string) (gokv.Store, error) {
			return syncmap.NewStore(syncmap.Options{Codec: encoding.Gob}), nil
		},
		Parallelism:       uint8(4),
		Filter:            DefaultFilterOptions(),
		Headless:          DefaultHeadlessOptions(),
		MangaDex:          DefaultMangaDexOptions(),
		MangaPlus:         DefaultMangaPlusOptions(),
		MangaPlusCreators: DefaultMangaPlusCreatorsOptions(),
	}
}
