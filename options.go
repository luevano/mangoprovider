package mangoprovider

import (
	"fmt"
	"net/http"

	"github.com/luevano/mangodex"
	"github.com/luevano/mangoplus"
	"github.com/luevano/mangoplus/creators"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
)

const BundleID = "mango"

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36"

// FilterOptions are general filtering options for all sources.
type FilterOptions struct {
	// NSFW if the manga results should contain NSFW content.
	//
	// Only if the source allows to filter by NSFW.
	NSFW bool

	// Language of the manga results.
	//
	// Only if the source allows to filter by language.
	Language string

	// AvoidDuplicateChapters will not include duplicate
	// chapters in the results.
	AvoidDuplicateChapters bool

	// ShowUnavailableChapters will not show unavailable
	// chapters in the results.
	//
	// Such as external sites for the chapters
	// (in the case of MangaDex that points to MangaPlus for example).
	ShowUnavailableChapters bool
}

func (f *FilterOptions) String() string {
	return fmt.Sprintf(
		"filters[%t&%s&%t&%t]",
		f.NSFW,
		f.Language,
		f.AvoidDuplicateChapters,
		f.ShowUnavailableChapters,
	)
}

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

	// Parallelism to use when making HTTP requests.
	Parallelism uint8

	// Filter options.
	Filter FilterOptions

	// Headless options.
	Headless HeadlessOptions

	// MangaDex options
	MangaDex MangaDexOptions

	// MangaPlus options.
	MangaPlus MangaPlusOptions

	// MangaPlusCreators options.
	MangaPlusCreators MangaPlusCreatorsOptions
}

func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{},
		UserAgent:  defaultUserAgent,
		CacheStore: func(dbName, bucketName string) (gokv.Store, error) {
			return syncmap.NewStore(syncmap.DefaultOptions), nil
		},
		Parallelism:       uint8(15),
		Filter:            DefaultFilterOptions(),
		Headless:          DefaultHeadlessOptions(),
		MangaDex:          DefaultMangaDexOptions(),
		MangaPlus:         DefaultMangaPlusOptions(),
		MangaPlusCreators: DefaultMangaPlusCreatorsOptions(),
	}
}
