package mangathemesia

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var FlamecomicsInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-flamecomics",
	Name:        "FlameComics",
	Version:     "0.3.0",
	Description: "FlameComics scraper",
	Website:     "https://flamecomics.com/",
}

// FlameComics in tachiyomi (RIP) has a really complicated
// logic for dealing with "composite images" whatever that means, gotta keep an eye
var FlamecomicsConfig = flamecomics()

func flamecomics() *scraper.Configuration {
	a := Mangathemesia(FlamecomicsInfo.ID, FlamecomicsInfo.Website, "series")

	// Need to remove trash cover URL, contains a prefixed URL
	// that basically "converts" the original one?
	a.MangaExtractor.Cover = flamecomicsMangaExtractorCover
	a.MangaByIDExtractor.Cover = flamecomicsMangaExtractorCover

	return a
}

func flamecomicsMangaExtractorCover(selection *goquery.Selection) string {
	imgURL := selection.Find("img").AttrOr("src", "")
	// assuming that the extension lies at the end
	if strings.HasSuffix(imgURL, ".webp") {
		// no support for webp, specially because it's used for
		// videos/gifs lol
		return ""
	}
	return fmt.Sprintf("http%s", imgURL)
}
