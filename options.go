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

type HeadlessOptions struct {
	UseFlaresolverr bool
	FlaresolverrURL string
}

func DefaultHeadlessOptions() HeadlessOptions {
	return HeadlessOptions{
		UseFlaresolverr: false,
		FlaresolverrURL: "http://localhost:8191/v1",
	}
}

type FilterOptions struct {
	NSFW                    bool
	Language                string
	TitleChapterNumber      bool
	AvoidDuplicateChapters  bool
	ShowUnavailableChapters bool
}

func DefaultFilterOptions() FilterOptions {
	return FilterOptions{
		NSFW:                    false,
		Language:                "en",
		TitleChapterNumber:      false,
		AvoidDuplicateChapters:  true,
		ShowUnavailableChapters: false,
	}
}

func (f *FilterOptions) String() string {
	return fmt.Sprintf(
		"filters[%t&%s&%t&%t&%t]",
		f.NSFW,
		f.Language,
		f.TitleChapterNumber,
		f.AvoidDuplicateChapters,
		f.ShowUnavailableChapters,
	)
}

type MangaDexOptions struct {
	mangodex.Options
	DataSaver bool
}

func DefaultMangaDexOptions() MangaDexOptions {
	o := mangodex.DefaultOptions()
	return MangaDexOptions{
		Options:   o,
		DataSaver: false,
	}
}

type MangaPlusOptions struct {
	mangoplus.Options
	Quality string
}

func DefaultMangaPlusOptions() MangaPlusOptions {
	o := mangoplus.DefaultOptions()
	return MangaPlusOptions{
		Options: o,
		Quality: "super_high",
	}
}

type MangaPlusCreatorsOptions struct {
	creators.Options
}

func DefaultMangaPlusCreatorsOptions() MangaPlusCreatorsOptions {
	o := creators.DefaultOptions()
	return MangaPlusCreatorsOptions{
		Options: o,
	}
}

type Options struct {
	HTTPClient        *http.Client
	UserAgent         string
	HTTPStore         func(providerID string) (gokv.Store, error)
	Parallelism       uint8
	Filter            FilterOptions
	Headless          HeadlessOptions
	MangaDex          MangaDexOptions
	MangaPlus         MangaPlusOptions
	MangaPlusCreators MangaPlusCreatorsOptions
}

func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{},
		UserAgent:  defaultUserAgent,
		HTTPStore: func(providerID string) (gokv.Store, error) {
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
