package mangoprovider

import (
	"fmt"
	"net/http"

	"github.com/philippgille/gokv"
)

const (
	BundleID  = "mango"
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"
)

type Headless struct {
	UseFlaresolverr bool
	FlaresolverrURL string
}

type Filter struct {
	NSFW                    bool
	Language                string
	MangaDexDataSaver       bool
	TitleChapterNumber      bool
	AvoidDuplicateChapters  bool
	ShowUnavailableChapters bool
}

func (f *Filter) String() string {
	return fmt.Sprintf(
		"loaderOptions[%t&%s&%t&%t&%t&%t]",
		f.NSFW,
		f.Language,
		f.MangaDexDataSaver,
		f.TitleChapterNumber,
		f.AvoidDuplicateChapters,
		f.ShowUnavailableChapters,
	)
}

// TODO: include parallelism option
type Options struct {
	HTTPClient        *http.Client
	HTTPStoreProvider func(providerID string) (gokv.Store, error)
	Headless
	Filter
}
