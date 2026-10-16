package toonily

import (
	"regexp"
	"strings"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
	"github.com/luevano/mangoprovider/scrapers/madara"
)

var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-toonily",
	Name:        "Toonily",
	Version:     "0.1.6",
	Description: "Toonily scraper",
	Website:     "https://toonily.com/",
}

// titleSpecialCharacters: keiyoushi's Toonily searches without them.
var titleSpecialCharacters = regexp.MustCompile(`[^a-z0-9]+`)

// Config: A new configuration, built on every load: keiyoushi's src/en/toonily on Madara (no
// rateLimit there either, the scraper's default ~4/s).
func Config(_ *mango.Logger) *scraper.Configuration {
	return madara.Config(madara.Options{
		ID:              Info.ID,
		BaseURL:         Info.Website,
		ScanlationGroup: Info.Name,
		IDCode:          "tl",
		// /webtoon/<slug>/ redirects here
		MangaSubString: "serie",
		ChapterMode:    madara.MangaAjax,
		// webtoons, chapters only
		SingleVolume: true,
		KeepNonManga: true,
		SearchQuery: func(query string) string {
			return strings.TrimSpace(titleSpecialCharacters.ReplaceAllString(strings.ToLower(query), " "))
		},
		DateLayouts: []string{"Jan 2, 06"},
		Selectors: madara.Selectors{
			Description: "div.content-area div.summary__content",
			SearchCard:  "div.page-item-detail.manga",
		},
		Headers: map[string]string{"Cookie": "toonily-mature=1"},
	})
}
