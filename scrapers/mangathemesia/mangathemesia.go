// Package mangathemesia builds the scraper configuration of a site on the MangaThemesia
// WordPress theme (formerly WPMangaStream / WPMangaReader), following keiyoushi's
// lib-multisrc/mangathemesia (MangaThemesia.kt, lib 1.6): a site is its base url and what it
// does differently (Options).
package mangathemesia

import (
	"encoding/base64"
	"encoding/json"
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
	"github.com/luevano/mangoprovider/fetch"
	"github.com/luevano/mangoprovider/scraper"
	"golang.org/x/time/rate"
)

// keiyoushi's defaults (MangaThemesia.kt)
const (
	defaultMangaDir   = "manga"
	defaultDateLayout = "January 2, 2006"

	searchMangaSelector   = ".utao .uta .imgu, .listupd .bs .bsx, .listo .bs .bsx"
	nextPageSelector      = "div.pagination .next, div.hpage .r"
	seriesDetailsSelector = "div.bigcontent, div.animefull, div.main-info, div.postbody"
	seriesTitleSelector   = ".entry-title, .ts-breadcrumb li:last-child span"
	descriptionSelector   = ".desc, .entry-content[itemprop=description]"
	thumbnailSelector     = ".infomanga > div[itemprop=image] img, .thumb img"
	chapterListSelector   = "div.bxcl li, div.cl li, #chapterlist li, ul li:has(div.chbox):has(div.eph-num)"
	chapterNameSelector   = ".lch a, .chapternum"
	chapterDateSelector   = ".chapterdate"
	pageSelector          = "div#readerarea img"
	updatedOnLayout       = "2006-01-02"
	updatedOnSelector     = ".listinfo time[itemprop=dateModified], .fmed:contains(update) time, span:contains(update) time"
)

// selectorFor: keiyoushi's selector(): selector with "%s" as each of the words (quoted, cascadia
// wants a string for words with spaces), joined.
func selectorFor(selector string, words ...string) string {
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = strings.ReplaceAll(selector, "%s", strconv.Quote(w))
	}
	return strings.Join(parts, ", ")
}

var (
	artistSelector = selectorFor(".infotable tr:contains(%s) td:last-child, .tsinfo .imptdt:contains(%s) i, .fmed b:contains(%s)+span, span:contains(%s)",
		"artist", "Artiste", "Artista", "الرسام", "الناشر", "İllüstratör", "Çizer", "Sanatçı", "นักวาด")
	authorSelector = selectorFor(".infotable tr:contains(%s) td:last-child, .tsinfo .imptdt:contains(%s) i, .fmed b:contains(%s)+span, span:contains(%s)",
		"Author", "Auteur", "autor", "المؤلف", "Mangaka", "seniman", "Pengarang", "Yazar", "ผู้แต่ง", "นักเขียน")
	altNameSelector = ".alternative, .wd-full:contains(alt) span, .alter, .seriestualt, " +
		selectorFor(".infotable tr:contains(%s) td:last-child", "Alternative", "Alternatif", "الأسماء الثانوية")
	genreSelector = "div.gnr a, .mgen a, .seriestugenre a, " + selectorFor("span:contains(%s)", "genre", "التصنيف")
	typeSelector  = selectorFor(".infotable tr:contains(%s) td:last-child, .tsinfo .imptdt:contains(%s) i, .tsinfo .imptdt:contains(%s) a, .fmed b:contains(%s)+span, span:contains(%s) a",
		"type", "ประเภท", "النوع", "tipe", "Türü") + ", a[href*=type\\=]"
	// ours, not keiyoushi's (like madara's Release)
	releaseSelector = selectorFor(".infotable tr:contains(%s) td:last-child, .tsinfo .imptdt:contains(%s) i, .fmed b:contains(%s)+span",
		"Released")
	statusSelector = selectorFor(".infotable tr:contains(%s) td:last-child, .tsinfo .imptdt:contains(%s) i, .fmed b:contains(%s)+span span:contains(%s)",
		"status", "Statut", "Durum", "連載状況", "Estado", "الحالة", "حالة العمل", "สถานะ", "stato", "Statüsü")
)

// Options: A MangaThemesia site, what isn't keiyoushi's default (its overrides of the same name).
type Options struct {
	// ID: The provider's id, e.g. "mango-asurascans".
	ID string
	// BaseURL: The site, ending with "/".
	BaseURL string
	// ScanlationGroup: Optional, the chapters' group (e.g. the site's name).
	ScanlationGroup string
	// IDCode: The metadata's provider id code, see scraper.MetadataExtractor.
	IDCode string

	// MangaDir: keiyoushi's mangaUrlDirectory without slashes, "manga" by default
	// (/manga/<slug>/, also the search's path).
	MangaDir string
	// TitleFilter: Search with keiyoushi's /<MangaDir>/?title=<query>&page=<n> instead of
	// WordPress' /?s=<query>: not every site filters by title there (checked 2026-10-16:
	// asterscans does, kingofshojo and ravenscans list everything), all of them have ?s=.
	TitleFilter bool
	// DateLayouts: The chapters' dates (time.Parse layouts), "January 2, 2006" by default
	// (keiyoushi's datePattern "MMMM d, yyyy"). Relative ones ("2 days ago") always parse.
	DateLayouts []string

	// Selectors: Optional, what's different from keiyoushi's.
	Selectors Selectors

	// Headers: Optional, sent with every html request (e.g. a cookie).
	Headers map[string]string
	// RateLimit, RateBurst: Optional, see scraper.Configuration.
	RateLimit rate.Limit
	RateBurst int
	// Fetch: Optional, see scraper.Configuration.
	Fetch fetch.Strategy
}

// Selectors: keiyoushi's selectors of the same name (Release is ours), empty is the default.
type Selectors struct {
	SearchManga, NextPage                                                   string
	SeriesDetails, Title, Artist, Author, Description, AltName, Genre, Type string
	Status, Thumbnail, Release                                              string
	ChapterList, ChapterName, ChapterDate                                   string
	Page                                                                    string
}

func (o Options) withDefaults() Options {
	or := func(s *string, d string) {
		if *s == "" {
			*s = d
		}
	}
	o.MangaDir = strings.Trim(o.MangaDir, "/")
	or(&o.MangaDir, defaultMangaDir)
	if len(o.DateLayouts) == 0 {
		o.DateLayouts = []string{defaultDateLayout}
	}
	s := &o.Selectors
	or(&s.SearchManga, searchMangaSelector)
	or(&s.NextPage, nextPageSelector)
	or(&s.SeriesDetails, seriesDetailsSelector)
	or(&s.Title, seriesTitleSelector)
	or(&s.Artist, artistSelector)
	or(&s.Author, authorSelector)
	or(&s.Description, descriptionSelector)
	or(&s.AltName, altNameSelector)
	or(&s.Genre, genreSelector)
	or(&s.Type, typeSelector)
	or(&s.Status, statusSelector)
	or(&s.Thumbnail, thumbnailSelector)
	or(&s.Release, releaseSelector)
	or(&s.ChapterList, chapterListSelector)
	or(&s.ChapterName, chapterNameSelector)
	or(&s.ChapterDate, chapterDateSelector)
	or(&s.Page, pageSelector)
	return o
}

// Mangathemesia: The configuration of the site in info (its id, name as the scanlation group
// and website) with its mangas under mangaDir, everything else keiyoushi's default.
func Mangathemesia(info libmangal.ProviderInfo, mangaDir string) *scraper.Configuration {
	return Config(Options{
		ID:              info.ID,
		BaseURL:         info.Website,
		ScanlationGroup: info.Name,
		MangaDir:        mangaDir,
	})
}

// Config: The site's scraper configuration, built from keiyoushi's MangaThemesia.
func Config(o Options) *scraper.Configuration {
	o = o.withDefaults()
	sel := o.Selectors
	config := &scraper.Configuration{
		Name: o.ID,
		// no rateLimit in keiyoushi's lib-multisrc/mangathemesia, the scraper's default ~4/s
		// unless the site has one (e.g. src/en/mangatx rateLimit(3))
		Delay:            50 * time.Millisecond,
		RateLimit:        o.RateLimit,
		RateBurst:        o.RateBurst,
		Fetch:            o.Fetch,
		ReverseChapters:  true,
		BaseURL:          o.BaseURL,
		Headers:          o.Headers,
		SearchPagination: &scraper.Pagination{NextSelector: sel.NextPage},
		GenerateSearchByIDURL: func(baseURL, id string) (string, error) {
			return baseURL + o.MangaDir + "/" + id + "/", nil
		},
		MangaByIDExtractor: &scraper.MangaByIDExtractor{
			Selector: sel.SeriesDetails,
			Title: func(s *goquery.Selection) string {
				return s.Find(sel.Title).First().Text()
			},
			Cover: func(s *goquery.Selection) string {
				return ImageURL(s.Find(sel.Thumbnail).First())
			},
		},
		MetadataExtractor: &scraper.MetadataExtractor{
			IDCode:            o.IDCode,
			Extract:           o.extractMetadata,
			StartFromChapters: true,
		},
		MangaExtractor: &scraper.MangaExtractor{
			Selector: sel.SearchManga,
			Title: func(s *goquery.Selection) string {
				return s.Find("a").First().AttrOr("title", "")
			},
			URL: func(s *goquery.Selection) string {
				return s.Find("a").First().AttrOr("href", "")
			},
			Cover: func(s *goquery.Selection) string {
				return ImageURL(s.Find("img").First())
			},
			ID: o.mangaID,
		},
		VolumeExtractor: &scraper.VolumeExtractor{
			Selector: "body",
			Number:   func(*goquery.Selection) float32 { return 1 },
		},
		ChapterExtractor: &scraper.ChapterExtractor{
			Selector: sel.ChapterList,
			Title: func(s *goquery.Selection) string {
				if name := mango.CleanString(s.Find(sel.ChapterName).Text()); name != "" {
					return name
				}
				return s.Find("a").First().Text()
			},
			URL: func(s *goquery.Selection) string {
				return s.Find("a").First().AttrOr("href", "")
			},
			ID: o.chapterID,
			Date: func(s *goquery.Selection) metadata.Date {
				return o.chapterDate(s, time.Now())
			},
		},
		PageExtractor: &scraper.PageExtractor{
			// the whole page: the images, else the ts_reader script's
			Selector: "html",
			URLs:     o.pageURLs,
		},
	}
	if o.TitleFilter {
		config.SearchRequest = func(baseURL, query string, page int) (scraper.Request, error) {
			return o.titleSearch(baseURL, query, page)
		}
	} else {
		// the next pages are the next links, /page/<n>/?s=<query>
		config.GenerateSearchURL = func(baseURL, query string) (string, error) {
			return baseURL + "?" + url.Values{"s": {query}}.Encode(), nil
		}
	}
	if o.ScanlationGroup != "" {
		config.ChapterExtractor.ScanlationGroup = func(*goquery.Selection) string { return o.ScanlationGroup }
	}
	return config
}

// titleSearch: keiyoushi's searchMangaUrl: /<MangaDir>/?title=<query>&page=<n>.
func (o Options) titleSearch(baseURL, query string, page int) (scraper.Request, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return scraper.Request{}, err
	}
	u = u.JoinPath(o.MangaDir)
	u.Path += "/"
	params := url.Values{"page": {strconv.Itoa(page)}}
	if query != "" {
		params.Set("title", query)
	}
	u.RawQuery = params.Encode()
	return scraper.Request{URL: u.String()}, nil
}

// mangaID: The slug after MangaDir in a manga url, /<MangaDir>/<slug>/.
func (o Options) mangaID(rawURL string) string {
	parts := scraper.PathParts(rawURL)
	for i, part := range parts {
		if part == o.MangaDir && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	// a site that moved its mangas
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// chapterID: The chapter's slug, the last part of its url (/<slug>/ at the root). A locked
// chapter without a link resolves to the manga page: no id, skipped (keiyoushi's sites
// filter those out too, e.g. src/en/athreascans).
func (o Options) chapterID(rawURL string) string {
	parts := scraper.PathParts(rawURL)
	if len(parts) == 0 || (len(parts) == 2 && parts[0] == o.MangaDir) {
		return ""
	}
	return parts[len(parts)-1]
}

// chapterDate: keiyoushi's chapterFromElement's date: the .chapterdate text, none if it doesn't
// parse. The newest chapter (the first) without one gets the manga's "Updated On" date, like
// keiyoushi's chapterListParse.
func (o Options) chapterDate(s *goquery.Selection, now time.Time) metadata.Date {
	if date, ok := mango.ParseDate(s.Find(o.Selectors.ChapterDate).First().Text(), now, o.DateLayouts...); ok {
		return date
	}
	root := s.Closest("html")
	if first := root.Find(o.Selectors.ChapterList).First(); first.Length() == 0 || first.Get(0) != s.Get(0) {
		return metadata.Date{}
	}
	updated := strings.TrimSpace(root.Find(updatedOnSelector).First().AttrOr("datetime", ""))
	if len(updated) < len(updatedOnLayout) {
		return metadata.Date{}
	}
	if t, err := time.Parse(updatedOnLayout, updated[:len(updatedOnLayout)]); err == nil {
		return mango.DateOf(t)
	}
	return metadata.Date{}
}

var (
	// keiyoushi's JSON_IMAGE_LIST_REGEX
	imageListRegex = regexp.MustCompile(`["']?(?:images|imageUrls)["']?\s*[:=]\s*(\[.*?\])`)
	// "ts_reader.run({" in base64
	tsReaderScript = "script[src^='data:text/javascript;base64,dHNfcmVhZGVyLnJ1bih7']"
)

// pageURLs: keiyoushi's pageListParse: the page's images, else the list in the ts_reader script
// (inline, or a base64 data: script).
func (o Options) pageURLs(doc *goquery.Selection) []string {
	var urls []string
	doc.Find(o.Selectors.Page).Each(func(_ int, img *goquery.Selection) {
		if u := ImageURL(img); u != "" {
			urls = append(urls, u)
		}
	})
	if len(urls) > 0 {
		return urls
	}

	var script string
	if src, ok := doc.Find(tsReaderScript).First().Attr("src"); ok {
		_, data, _ := strings.Cut(src, "base64,")
		if decoded, err := base64.StdEncoding.DecodeString(data); err == nil {
			script = string(decoded)
		}
	}
	if script == "" {
		doc.Find("script").Each(func(_ int, s *goquery.Selection) {
			if script == "" && imageListRegex.MatchString(s.Text()) {
				script = s.Text()
			}
		})
	}
	m := imageListRegex.FindStringSubmatch(script)
	if m == nil {
		return nil
	}
	var images []string
	if err := json.Unmarshal([]byte(m[1]), &images); err != nil {
		return nil
	}
	return images
}

// ImageURL: keiyoushi's imgAttr: data-lazy-src, data-src, data-cfsrc or src.
func ImageURL(img *goquery.Selection) string {
	for _, attr := range []string{"data-lazy-src", "data-src", "data-cfsrc", "src"} {
		if v, ok := img.Attr(attr); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// ownText: The text of s's own text nodes, not its children's (jsoup's ownText).
func ownText(s *goquery.Selection) string {
	var sb strings.Builder
	s.Contents().Each(func(_ int, c *goquery.Selection) {
		if goquery.NodeName(c) == "#text" {
			sb.WriteString(c.Text())
		}
	})
	return mango.CleanString(sb.String())
}

// removeEmptyPlaceholder: keiyoushi's removeEmptyPlaceholder.
func removeEmptyPlaceholder(s string) string {
	switch s {
	case "-", "N/A", "n/a", "Unknown":
		return ""
	}
	return s
}

var altNameSepRegex = regexp.MustCompile(`[|/•,;]`)

// extractMetadata: keiyoushi's mangaDetailsParse, the series type as the format instead of a
// genre, plus the release year.
func (o Options) extractMetadata(page *goquery.Selection, meta *mangadata.Metadata) {
	sel := o.Selectors
	details := page.Find(sel.SeriesDetails).First()
	if details.Length() == 0 {
		return
	}
	meta.EnglishTitle = mango.CleanString(details.Find(sel.Title).First().Text())
	if artist := removeEmptyPlaceholder(ownText(details.Find(sel.Artist).First())); artist != "" {
		meta.ArtistList = []string{artist}
	}
	if author := removeEmptyPlaceholder(ownText(details.Find(sel.Author).First())); author != "" {
		meta.AuthorList = []string{author}
	}
	if description := details.Find(sel.Description).First(); description.Find("p").Length() > 0 {
		meta.Summary = strings.Join(scraper.Texts(description.Find("p")), "\n\n")
	} else {
		meta.Summary = mango.CleanString(description.Text())
	}
	if alt := ownText(details.Find(sel.AltName).First()); alt != "" {
		meta.Synonyms = altNameSepRegex.Split(alt, -1)
	}
	meta.GenreList = scraper.Texts(details.Find(sel.Genre))
	if typ := ownText(details.Find(sel.Type).First()); typ != "" {
		meta.PublicationFormat, meta.CountryOfOrigin = typ, mango.CountryOfType(typ)
	}
	if status := details.Find(sel.Status).First(); status.Length() > 0 {
		meta.PublicationStatus = Status(status.Text())
	}
	meta.CoverImage = ImageURL(details.Find(sel.Thumbnail).First())
	meta.DateStart = scraper.YearDate(details.Find(sel.Release).First().Text())
}

// keiyoushi's parseStatus words, checked in this order
var statuses = []struct {
	status metadata.Status
	words  []string
}{
	{metadata.StatusReleasing, []string{"مستمرة", "en curso", "ongoing", "on going", "new season", "mass released", "ativo", "en cours", "en cours de publication", "đang tiến hành", "em lançamento", "онгоінг", "publishing", "devam ediyor", "em andamento", "in corso", "güncel", "berjalan", "продолжается", "updating", "lançando", "in arrivo", "emision", "en emision", "مستمر", "curso", "en marcha", "publicandose", "publicando", "连载中", "devam etmekte", "連載中"}},
	{metadata.StatusFinished, []string{"completed", "completo", "complété", "fini", "achevé", "terminé", "tamamlandı", "đã hoàn thành", "hoàn thành", "مكتملة", "завершено", "finished", "finalizad", "completata", "one-shot", "bitti", "tamat", "completado", "concluído", "完結", "concluido", "已完结", "bitmiş"}},
	{metadata.StatusCancelled, []string{"canceled", "cancelled", "cancelado", "cancellato", "cancelados", "dropped", "discontinued", "abandonné"}},
	{metadata.StatusHiatus, []string{"hiatus", "on hold", "season end", "pausado", "en espera", "en pause", "en attente", "hiato"}},
}

// Status: The metadata status of a status text, "UNKNOWN" if it's none of keiyoushi's.
func Status(text string) metadata.Status {
	text = strings.ToLower(text)
	for _, s := range statuses {
		for _, word := range s.words {
			if strings.Contains(text, word) {
				return s.status
			}
		}
	}
	return metadata.Status("UNKNOWN")
}
