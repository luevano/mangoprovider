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

type Options struct {
	HTTPClient  *http.Client
	HTTPStore   func(providerID string) (gokv.Store, error)
	Parallelism uint8
	Headless
	Filter
}
