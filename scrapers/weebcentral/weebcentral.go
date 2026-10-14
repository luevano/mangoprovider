package weebcentral

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
	"golang.org/x/time/rate"
)

// searchPageSize: Results in a search page.
const searchPageSize = 32

// MangaSee shut down and moved to Weeb Central; its id is in the history so it still resolves.
var Info = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-weebcentral",
	Name:        "Weeb Central",
	Version:     "0.4.4",
	Description: "Weeb Central scraper",
	Website:     "https://weebcentral.com/",
	History: []libmangal.ProviderAlias{
		{ID: mango.BundleID + "-mangasee", Name: "MangaSee", Website: "https://mangasee123.com/"},
	},
}

// Config: A new configuration, built on every load.
func Config(log *mango.Logger) *scraper.Configuration {
	return &scraper.Configuration{
		Name:            Info.ID,
		Delay:           50 * time.Millisecond,
		ReverseChapters: true,
		// keiyoushi's rateLimit(1, 2.seconds) on the site's host (src/en/weebcentral/.../WeebCentral.kt),
		// images are on other hosts
		RateLimit: rate.Every(2 * time.Second),
		RateBurst: 1,
		BaseURL:   Info.Website,
		GenerateSearchURL: func(baseUrl string, query string) (string, error) {
			params := url.Values{}
			params.Set("text", query)
			params.Set("sort", "Best Match")
			params.Set("order", "Descending")
			params.Set("official", "Any")
			params.Set("anime", "Any")
			params.Set("adult", "Any")
			params.Set("display_mode", "Full Display")
			return baseUrl + "search/data?" + params.Encode(), nil
		},
		// keiyoushi's: 32 per page (whatever limit says), a "View More Results..." button
		// when there are more
		SearchPagination: &scraper.Pagination{
			NextSelector: "button",
			URL: func(searchURL string, page int) (string, error) {
				u, err := url.Parse(searchURL)
				if err != nil {
					return "", err
				}
				params := u.Query()
				params.Set("limit", strconv.Itoa(searchPageSize))
				params.Set("offset", strconv.Itoa((page-1)*searchPageSize))
				u.RawQuery = params.Encode()
				return u.String(), nil
			},
		},
		GenerateSearchByIDURL: func(baseUrl string, id string) (string, error) {
			return baseUrl + "series/" + id, nil
		},
		// the series page only shows the latest chapters
		ChaptersURL: func(manga mango.Manga) string {
			return Info.Website + "series/" + manga.ID + "/full-chapter-list"
		},
		// the chapter page loads its images from here
		PagesURL: func(chapter mango.Chapter) string {
			return Info.Website + "chapters/" + chapter.ID + "/images?is_prev=False&current_page=1&reading_style=long_strip"
		},
		MangaByIDExtractor: &scraper.MangaByIDExtractor{
			Selector: "main",
			Title: func(selection *goquery.Selection) string {
				return selection.Find("h1").First().Text()
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("picture img").First().AttrOr("src", "")
			},
		},
		MetadataExtractor: &scraper.MetadataExtractor{
			IDCode:  "wc",
			Extract: extractMetadata,
		},
		MangaExtractor: &scraper.MangaExtractor{
			Selector: "article.bg-base-300",
			Title: func(selection *goquery.Selection) string {
				return selection.Find(`section:nth-child(2) a[href*="/series/"]`).First().Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.Find(`section:nth-child(2) a[href*="/series/"]`).First().AttrOr("href", "")
			},
			Cover: func(selection *goquery.Selection) string {
				return selection.Find("picture img").First().AttrOr("src", "")
			},
			// e.g. https://weebcentral.com/series/01J76XY7EF75DJNQCV04HTPDZK/Berserk
			ID: func(_url string) string {
				id, _ := pathAfter(_url, "series")
				return id
			},
		},
		VolumeExtractor: &scraper.VolumeExtractor{
			Selector: "body",
			Number: func(selection *goquery.Selection) float32 {
				return 1.0
			},
		},
		ChapterExtractor: &scraper.ChapterExtractor{
			Selector: `a[href*="/chapters/"]`,
			Title: func(selection *goquery.Selection) string {
				return selection.Find("span.grow > span").First().Text()
			},
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("href", "")
			},
			// e.g. https://weebcentral.com/chapters/01KX4BFP5E3FEZ50TRS752CT9V
			ID: func(_url string) string {
				id, _ := pathAfter(_url, "chapters")
				return id
			},
			Number: chapterNumber,
			Date: func(selection *goquery.Selection) metadata.Date {
				date, ok := mango.ParseDate(selection.Find("time").AttrOr("datetime", ""), time.Now(), time.RFC3339)
				if !ok {
					log.Log("chapter %q has no valid date", mango.CleanString(selection.Text()))
				}
				return date
			},
			ScanlationGroup: func(_ *goquery.Selection) string {
				return "Weeb Central"
			},
		},
		PageExtractor: &scraper.PageExtractor{
			Selector: `img[alt^="Page"]`,
			URL: func(selection *goquery.Selection) string {
				return selection.AttrOr("src", "")
			},
		},
	}
}

// chapterNumber: Prologues are numbered before chapter 1, e.g. "Prologue 3" is 0.03.
func chapterNumber(title string) (float32, bool) {
	if n, ok := strings.CutPrefix(title, "Prologue "); ok {
		if parsed, err := strconv.ParseFloat(n, 32); err == nil && parsed < 100 {
			return float32(parsed / 100), true
		}
	}
	return mango.ParseChapterNumber(title)
}

// pathAfter: The path segment after the given one, e.g. the id after "series".
func pathAfter(_url, segment string) (string, bool) {
	parts := scraper.PathParts(_url)
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == segment {
			return parts[i+1], true
		}
	}
	return "", false
}

// trackers: The ids of the series' trackers links that libmangal can bind.
var trackers = []struct {
	re     *regexp.Regexp
	code   metadata.IDCode
	source metadata.IDSource
}{
	{regexp.MustCompile(`anilist\.co/manga/(\d+)`), metadata.IDCodeAnilist, metadata.IDSourceAnilist},
	{regexp.MustCompile(`myanimelist\.net/manga/(\d+)`), metadata.IDCodeMyAnimeList, metadata.IDSourceMyAnimeList},
	{regexp.MustCompile(`mangaupdates\.com/series/([0-9a-z]+)`), metadata.IDCodeMangaUpdates, metadata.IDSourceMangaUpdates},
}

// extractMetadata: keiyoushi's getMangaDetails, plus the release year and the trackers' ids.
func extractMetadata(page *goquery.Selection, meta *mangadata.Metadata) {
	sections := page.Find("section[x-data] > section")
	left, right := sections.Eq(0), sections.Eq(1)
	detail := func(s *goquery.Selection, name string) *goquery.Selection {
		return s.Find("li").FilterFunction(func(_ int, li *goquery.Selection) bool {
			return strings.HasPrefix(mango.CleanString(li.ChildrenFiltered("strong").Text()), name)
		}).First()
	}

	meta.EnglishTitle = mango.CleanString(right.Find("h1").First().Text())
	if cover := left.Find("picture source").AttrOr("srcset", ""); cover != "" {
		meta.CoverImage = strings.Replace(cover, "small", "normal", 1)
	} else {
		meta.CoverImage = left.Find("picture img").AttrOr("src", "")
	}
	meta.AuthorList = scraper.Texts(detail(left, "Author").Find("span > a"))
	meta.GenreList = scraper.Texts(detail(left, "Tag").Find("a"))
	typ := mango.CleanString(detail(left, "Type").Find("a").Text())
	meta.PublicationFormat, meta.CountryOfOrigin = typ, mango.CountryOfType(typ)
	meta.PublicationStatus = mango.ParseStatus(detail(left, "Status").Find("a").Text())
	meta.DateStart = scraper.YearDate(detail(left, "Released").Find("span").Text())
	meta.Synonyms = scraper.Texts(detail(right, "Associated Name").Find("li"))
	meta.Summary = strings.TrimSpace(detail(right, "Description").Find("p").First().Text())

	detail(left, "Track").Find("span[data-tip] > a").Each(func(_ int, a *goquery.Selection) {
		href := a.AttrOr("href", "")
		for _, t := range trackers {
			m := t.re.FindStringSubmatch(href)
			switch {
			case m == nil:
			case t.source == metadata.IDSourceMangaUpdates:
				if id, ok := metadata.MangaUpdatesID(m[1]); ok {
					meta.OtherIDs = append(meta.OtherIDs, id)
				}
			default:
				meta.OtherIDs = append(meta.OtherIDs, metadata.ID{Raw: m[1], Code: t.code, Source: t.source})
			}
		}
	})
}
