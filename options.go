package mangoprovider

import (
	"fmt"
	"net/http"

	"github.com/philippgille/gokv"
)

const BundleID = "mango"

type Headless struct {
	UseFlaresolverr bool
	FlaresolverrURL string
}

type Filter struct {
	NSFW                    bool
	Language                string
	MangaPlusQuality        string
	MangaDexDataSaver       bool
	TitleChapterNumber      bool
	AvoidDuplicateChapters  bool
	ShowUnavailableChapters bool
}

func (f *Filter) String() string {
	return fmt.Sprintf(
		"loaderOptions[%t&%s&%s&%t&%t&%t&%t]",
		f.NSFW,
		f.Language,
		f.MangaPlusQuality,
		f.MangaDexDataSaver,
		f.TitleChapterNumber,
		f.AvoidDuplicateChapters,
		f.ShowUnavailableChapters,
	)
}

type MangaPlusOptions struct {
	OSVersion  string
	AppVersion string
	AndroidID  string
}

type Options struct {
	HTTPClient  *http.Client
	UserAgent   string
	HTTPStore   func(providerID string) (gokv.Store, error)
	Parallelism uint8
	Headless
	Filter
	MangaPlus MangaPlusOptions
}
