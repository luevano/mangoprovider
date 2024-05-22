package mangathemesia

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

var AsurascansInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-asurascans",
	Name:        "AsuraScans",
	Version:     "0.3.0",
	Description: "AsuraScans scraper",
	Website:     "https://asuracomic.net/",
}

var AsurascansConfig = asurascans()

func asurascans() *scraper.Configuration {
	a := Mangathemesia(AsurascansInfo.ID, AsurascansInfo.Website, "manga")

	// Need to remove trash cover URL, contains a prefixed URL
	// that basically "converts" the original one?
	a.MangaExtractor.Cover = asurascansMangaExtractorCover
	a.MangaByIDExtractor.Cover = asurascansMangaExtractorCover

	return a
}

func asurascansMangaExtractorCover(selection *goquery.Selection) string {
	imgURL := selection.Find("img").AttrOr("src", "")
	imgURLSplit := strings.Split(imgURL, "http")
	imgURL = imgURLSplit[len(imgURLSplit)-1]
	return fmt.Sprintf("http%s", imgURL)
}
