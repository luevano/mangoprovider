// Package madara builds the scraper configuration of a site on the Madara WordPress theme,
// following keiyoushi's lib-multisrc/madara (Madara and MadaraNoAjax, lib 1.6): a site is its
// base url and what it does differently (Options).
package madara

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/fetch"
	"github.com/luevano/mangoprovider/scraper"
	"golang.org/x/time/rate"
)

// ChapterMode: Where the chapters are listed, keiyoushi's ChapterMode.
type ChapterMode int

const (
	// MangaPage: the manga's page (keiyoushi's default).
	MangaPage ChapterMode = iota
	// MangaAjax: a POST to <manga url>/ajax/chapters/ (the "new chapter endpoint").
	MangaAjax
	// AdminAjax: a POST to /wp-admin/admin-ajax.php, action manga_get_chapters with the manga's
	// post id (from its page).
	AdminAjax
	// MangaAjaxQuery: a POST to /index.php, maction get_chapters with the manga's slug.
	MangaAjaxQuery
)

// keiyoushi's defaults (MadaraBase.kt, Madara.kt, MadaraNoAjax.kt)
const (
	defaultMangaSubString = "manga"
	defaultPageSize       = 25
	defaultDateLayout     = "January 2, 2006"
	ajaxTemplate          = "madara-core/content/content-archive"

	archiveSelector       = "div.page-item-detail, .manga__item, .c-tabs-item__content"
	searchCardSelector    = ".c-tabs-item__content"
	archiveURLSelector    = ".post-title a"
	nextPageSelector      = "div.nav-previous, a.nextpostslink, #navigation-ajax"
	titleSelector         = "div.post-title h3, div.post-title h1, #manga-title > h1"
	authorSelector        = "div.author-content > a, div.manga-authors > a"
	artistSelector        = "div.artist-content > a"
	statusSelector        = "div.summary-content, div.summary-heading:contains(Status) + div"
	descriptionSelector   = "div.description-summary div.summary__content, div.summary_content div.post-content_item > h5 + div, div.summary_content div.manga-excerpt"
	thumbnailSelector     = "div.summary_image img"
	genreSelector         = "div.genres-content a"
	tagSelector           = "div.tags-content a"
	seriesTypeSelector    = ".post-content_item:contains(Type) .summary-content"
	altNameSelector       = ".post-content_item:contains(Alt) .summary-content"
	releaseSelector       = ".post-content_item:contains(Release) .summary-content"
	publisherSelector     = ".post-content_item:contains(Publisher) .summary-content"
	chapterListSelector   = "li.wp-manga-chapter"
	chapterURLSelector    = "a"
	chapterDateSelector   = "span.chapter-release-date"
	pageListParseSelector = "div.page-break, li.blocks-gallery-item, .reading-content .text-left:not(:has(.blocks-gallery-item))"
)

// Options: A Madara site, what isn't keiyoushi's default (its overrides of the same name).
type Options struct {
	// ID: The provider's id, e.g. "mango-toonily".
	ID string
	// BaseURL: The site, ending with "/".
	BaseURL string
	// ScanlationGroup: Optional, the chapters' group (e.g. the site's name).
	ScanlationGroup string
	// IDCode: The metadata's provider id code (e.g. "tl"), see scraper.MetadataExtractor.
	IDCode string

	// MangaSubString: The path of the mangas, "manga" by default (/manga/<slug>/).
	MangaSubString string
	// ChapterMode: Where the chapters are listed, the manga page by default.
	ChapterMode ChapterMode
	// NoAjax: The site has no ajax search (keiyoushi's MadaraNoAjax): /?s=<query>&post_type=wp-manga.
	NoAjax bool
	// KeepNonManga: keiyoushi's filterNonMangaItems = false: the search also lists the novels
	// and other non manga posts (they're left out by default).
	KeepNonManga bool
	// PageSize: The ajax search's page size, 25 by default.
	PageSize int
	// SearchQuery: Optional, the query as the site's search wants it.
	SearchQuery func(query string) string
	// DateLayouts: The chapters' dates (time.Parse layouts), "January 2, 2006" by default
	// (keiyoushi's chapterDateFormat "MMMM d, yyyy"). Relative ones ("2 days ago") always parse.
	DateLayouts []string

	// SingleVolume: Optional, see scraper.Configuration: only for a site checked to have no
	// volumes (the theme can group chapters by volume, flattened like keiyoushi does).
	SingleVolume bool

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

// Selectors: keiyoushi's selectors of the same name (Release and Publisher are ours), empty is
// the default.
type Selectors struct {
	Archive, SearchCard, ArchiveURL, NextPage                                              string
	Title, Author, Artist, Status, Description, Thumbnail, Genre, Tag, SeriesType, AltName string
	Release, Publisher                                                                     string
	ChapterList, ChapterURL, ChapterName, ChapterDate                                      string
	PageListParse                                                                          string
}

func (o Options) withDefaults() Options {
	or := func(s *string, d string) {
		if *s == "" {
			*s = d
		}
	}
	or(&o.MangaSubString, defaultMangaSubString)
	if o.PageSize <= 0 {
		o.PageSize = defaultPageSize
	}
	if len(o.DateLayouts) == 0 {
		o.DateLayouts = []string{defaultDateLayout}
	}
	s := &o.Selectors
	or(&s.Archive, archiveSelector)
	or(&s.SearchCard, searchCardSelector)
	or(&s.ArchiveURL, archiveURLSelector)
	or(&s.NextPage, nextPageSelector)
	or(&s.Title, titleSelector)
	or(&s.Author, authorSelector)
	or(&s.Artist, artistSelector)
	or(&s.Status, statusSelector)
	or(&s.Description, descriptionSelector)
	or(&s.Thumbnail, thumbnailSelector)
	or(&s.Genre, genreSelector)
	or(&s.Tag, tagSelector)
	or(&s.SeriesType, seriesTypeSelector)
	or(&s.AltName, altNameSelector)
	or(&s.Release, releaseSelector)
	or(&s.Publisher, publisherSelector)
	or(&s.ChapterList, chapterListSelector)
	or(&s.ChapterURL, chapterURLSelector)
	or(&s.ChapterDate, chapterDateSelector)
	or(&s.PageListParse, pageListParseSelector)
	return o
}

var ajaxHeaders = map[string]string{"X-Requested-With": "XMLHttpRequest"}

// Config: The site's scraper configuration, built from keiyoushi's Madara.
func Config(o Options) *scraper.Configuration {
	o = o.withDefaults()
	sel := o.Selectors
	config := &scraper.Configuration{
		Name: o.ID,
		// no rateLimit in keiyoushi's lib-multisrc/madara, the scraper's default ~4/s unless
		// the site has one
		Delay:           50 * time.Millisecond,
		RateLimit:       o.RateLimit,
		RateBurst:       o.RateBurst,
		Fetch:           o.Fetch,
		SingleVolume:    o.SingleVolume,
		ReverseChapters: true,
		BaseURL:         o.BaseURL,
		Headers:         o.Headers,
		GenerateSearchByIDURL: func(baseURL, id string) (string, error) {
			return baseURL + o.MangaSubString + "/" + id + "/", nil
		},
		MangaByIDExtractor: &scraper.MangaByIDExtractor{
			Selector: "body",
			Title: func(s *goquery.Selection) string {
				return OwnText(s.Find(sel.Title).First())
			},
			Cover: func(s *goquery.Selection) string {
				return ImageURL(s.Find(sel.Thumbnail).First())
			},
		},
		MetadataExtractor: &scraper.MetadataExtractor{
			IDCode:  o.IDCode,
			Extract: o.extractMetadata,
			// madara pages rarely say when it started
			StartFromChapters: true,
		},
		MangaExtractor: &scraper.MangaExtractor{
			Selector: sel.Archive,
			Title: func(s *goquery.Selection) string {
				return s.Find(sel.ArchiveURL).First().Text()
			},
			URL: func(s *goquery.Selection) string {
				return s.Find(sel.ArchiveURL).First().AttrOr("href", "")
			},
			Cover: func(s *goquery.Selection) string {
				return ImageURL(s.Find("img").First())
			},
			ID: func(u string) string {
				id, _ := o.ids(u)
				return id
			},
		},
		VolumeExtractor: &scraper.VolumeExtractor{
			Selector: "body",
			Number:   func(*goquery.Selection) float32 { return 1 },
		},
		ChapterExtractor: &scraper.ChapterExtractor{
			Selector: sel.ChapterList,
			Title: func(s *goquery.Selection) string {
				if sel.ChapterName != "" {
					if name := s.Find(sel.ChapterName).First(); name.Length() > 0 {
						return name.Text()
					}
				}
				return s.Find(sel.ChapterURL).First().Text()
			},
			URL: func(s *goquery.Selection) string {
				return s.Find(sel.ChapterURL).First().AttrOr("href", "")
			},
			ID: func(u string) string {
				_, id := o.ids(u)
				return id
			},
			Date: func(s *goquery.Selection) metadata.Date {
				return o.chapterDate(s, time.Now())
			},
		},
		PageExtractor: &scraper.PageExtractor{
			Selector: sel.PageListParse,
			URL: func(s *goquery.Selection) string {
				return ImageURL(s.Find("img").First())
			},
		},
	}
	if o.ScanlationGroup != "" {
		config.ChapterExtractor.ScanlationGroup = func(*goquery.Selection) string { return o.ScanlationGroup }
	}

	if o.NoAjax {
		config.MangaExtractor.Selector = sel.Archive + ", " + sel.SearchCard
		config.SearchRequest = func(baseURL, query string, page int) (scraper.Request, error) {
			return o.htmlSearch(baseURL, query, page)
		}
		config.SearchPagination = &scraper.Pagination{NextSelector: sel.NextPage}
	} else {
		config.SearchRequest = func(baseURL, query string, page int) (scraper.Request, error) {
			return o.ajaxSearch(baseURL, query, page), nil
		}
		config.SearchPagination = &scraper.Pagination{PageSize: o.PageSize}
	}

	switch o.ChapterMode {
	case MangaAjax:
		config.ChaptersRequest = func(manga mango.Manga, _ func() (*goquery.Selection, error)) (scraper.Request, error) {
			return scraper.Request{URL: strings.TrimSuffix(manga.URL, "/") + "/ajax/chapters/", Form: url.Values{}, Headers: ajaxHeaders}, nil
		}
	case AdminAjax:
		config.ChaptersRequest = func(manga mango.Manga, mangaPage func() (*goquery.Selection, error)) (scraper.Request, error) {
			page, err := mangaPage()
			if err != nil {
				return scraper.Request{}, err
			}
			id := PostID(page)
			if id == "" {
				return scraper.Request{}, fmt.Errorf("no post id in %s", manga.URL)
			}
			form := url.Values{"action": {"manga_get_chapters"}, "manga": {id}}
			return scraper.Request{URL: o.BaseURL + "wp-admin/admin-ajax.php", Form: form, Headers: ajaxHeaders}, nil
		}
	case MangaAjaxQuery:
		config.ChaptersRequest = func(manga mango.Manga, _ func() (*goquery.Selection, error)) (scraper.Request, error) {
			slug, _ := o.ids(manga.URL)
			form := url.Values{"manga-core": {slug}, "manga_ajax": {"1"}, "maction": {"get_chapters"}}
			return scraper.Request{URL: o.BaseURL + "index.php", Form: form, Headers: ajaxHeaders}, nil
		}
	}
	return config
}

// ajaxSearch: keiyoushi's Madara.ajaxList for a search: page n of madara_load_more.
func (o Options) ajaxSearch(baseURL, query string, page int) scraper.Request {
	if o.SearchQuery != nil {
		query = o.SearchQuery(query)
	}
	form := url.Values{
		"action":                           {"madara_load_more"},
		"page":                             {strconv.Itoa(page - 1)},
		"template":                         {ajaxTemplate},
		"vars[paged]":                      {"1"},
		"vars[template]":                   {"archive"},
		"vars[posts_per_page]":             {strconv.Itoa(o.PageSize)},
		"vars[post_type]":                  {"wp-manga"},
		"vars[post_status]":                {"publish"},
		"vars[manga_archives_item_layout]": {"big_thumbnail"},
		"vars[s]":                          {query},
	}
	if !o.KeepNonManga {
		form.Set("vars[meta_query][0][key]", "_wp_manga_chapter_type")
		form.Set("vars[meta_query][0][value]", "manga")
	}
	return scraper.Request{URL: baseURL + "wp-admin/admin-ajax.php", Form: form, Headers: ajaxHeaders}
}

// htmlSearch: keiyoushi's MadaraNoAjax search: /page/<n>/?s=<query>&post_type=wp-manga.
func (o Options) htmlSearch(baseURL, query string, page int) (scraper.Request, error) {
	if o.SearchQuery != nil {
		query = o.SearchQuery(query)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return scraper.Request{}, err
	}
	if page > 1 {
		u = u.JoinPath("page", strconv.Itoa(page))
		u.Path += "/"
	}
	u.RawQuery = url.Values{"s": {query}, "post_type": {"wp-manga"}}.Encode()
	return scraper.Request{URL: u.String()}, nil
}

// ids: The manga and chapter ids (slugs) in a url, /<MangaSubString>/<manga>/<chapter>/.
// Without MangaSubString in the path (e.g. a site that moved its mangas), the first part is
// taken as it.
func (o Options) ids(rawURL string) (manga, chapter string) {
	parts := scraper.PathParts(rawURL)
	i := 0
	for j, part := range parts {
		if part == o.MangaSubString {
			i = j
			break
		}
	}
	if i+1 < len(parts) {
		manga = parts[i+1]
	}
	if i+2 < len(parts) {
		chapter = parts[i+2]
	}
	return manga, chapter
}

// chapterDate: keiyoushi's chapterFromElement: an image's alt, a link's title (new chapters,
// "2 days ago") or the date's text, none if neither parses.
func (o Options) chapterDate(s *goquery.Selection, now time.Time) metadata.Date {
	for _, text := range []string{
		s.Find("img:not(.thumb)").First().AttrOr("alt", ""),
		s.Find("span a").First().AttrOr("title", ""),
		s.Find(o.Selectors.ChapterDate).First().Text(),
	} {
		// the first that parses, keiyoushi's takes the first there (a "new" badge's alt)
		if date, ok := mango.ParseDate(text, now, o.DateLayouts...); ok {
			return date
		}
	}
	return metadata.Date{}
}

// OwnText: The text of s's own text nodes, not its children's (jsoup's ownText), e.g. a title
// without its badges.
func OwnText(s *goquery.Selection) string {
	var sb strings.Builder
	s.Contents().Each(func(_ int, c *goquery.Selection) {
		if goquery.NodeName(c) == "#text" {
			sb.WriteString(c.Text())
		}
	})
	return mango.CleanString(sb.String())
}

// ImageURL: keiyoushi's imageFromElement: the lazy loading attributes, the largest of srcset,
// or src.
func ImageURL(img *goquery.Selection) string {
	for _, attr := range []string{"data-src", "data-lazy-src", "data-lzl-src", "data-cfsrc", "data-manga-src"} {
		if v, ok := img.Attr(attr); ok {
			return strings.TrimSpace(v)
		}
	}
	if srcset, ok := img.Attr("srcset"); ok {
		if best := largestSrc(srcset); best != "" {
			return best
		}
	}
	return strings.TrimSpace(img.AttrOr("src", ""))
}

var descriptorRegex = regexp.MustCompile(`^(\d+(?:\.\d+)?)[wx]$`)

// largestSrc: The srcset's candidate with the largest descriptor, else the last one.
func largestSrc(srcset string) string {
	var best string
	bestSize := -1.0
	for _, candidate := range strings.Split(srcset, ",") {
		fields := strings.Fields(candidate)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "http") {
			continue
		}
		size := 0.0
		if len(fields) > 1 {
			if m := descriptorRegex.FindStringSubmatch(fields[1]); m != nil {
				size, _ = strconv.ParseFloat(m[1], 64)
			}
		}
		if size >= bestSize {
			best, bestSize = fields[0], size
		}
	}
	return best
}

// PostID: The manga's post id in its page (keiyoushi's mangaId), empty if it isn't there.
func PostID(page *goquery.Selection) string {
	for _, find := range []func() string{
		func() string { return page.Find("[id^=manga-chapters-holder]").First().AttrOr("data-id", "") },
		func() string { return page.Find("input.rating-post-id").First().AttrOr("value", "") },
		func() string { return page.Find("a[data-post]").First().AttrOr("data-post", "") },
		func() string {
			href := page.Find("link[rel=shortlink]").First().AttrOr("href", "")
			if u, err := url.Parse(href); err == nil {
				return u.Query().Get("p")
			}
			return ""
		},
	} {
		if id := strings.TrimSpace(find()); id != "" {
			return id
		}
	}
	return ""
}

var (
	updatingRegex   = regexp.MustCompile(`(?i)updating|atualizando`)
	altNameSepRegex = regexp.MustCompile(`\s*[;|]\s*`)
)

// extractMetadata: keiyoushi's parseDetails, plus the release year and the publisher. The
// alternative names are split on ";" or "|" (commas are in titles too).
func (o Options) extractMetadata(page *goquery.Selection, meta *mangadata.Metadata) {
	sel := o.Selectors
	notUpdating := func(texts []string) []string {
		var kept []string
		for _, t := range texts {
			if !updatingRegex.MatchString(t) {
				kept = append(kept, t)
			}
		}
		return kept
	}
	meta.EnglishTitle = OwnText(page.Find(sel.Title).First())
	meta.AuthorList = notUpdating(scraper.Texts(page.Find(sel.Author)))
	meta.ArtistList = notUpdating(scraper.Texts(page.Find(sel.Artist)))
	if description := page.Find(sel.Description).First(); description.Find("p").Length() > 0 {
		meta.Summary = strings.Join(scraper.Texts(description.Find("p")), "\n\n")
	} else {
		meta.Summary = mango.CleanString(description.Text())
	}
	if alt := OwnText(page.Find(sel.AltName).First()); alt != "" && !updatingRegex.MatchString(alt) {
		meta.Synonyms = altNameSepRegex.Split(alt, -1)
	}
	meta.CoverImage = ImageURL(page.Find(sel.Thumbnail).First())
	if status := page.Find(sel.Status).Last(); status.Length() > 0 {
		meta.PublicationStatus = Status(status.Text())
	}
	meta.GenreList = scraper.Texts(page.Find(sel.Genre))
	meta.TagList = scraper.Texts(page.Find(sel.Tag))
	if typ := mango.CleanString(page.Find(sel.SeriesType).First().Text()); typ != "" {
		meta.PublicationFormat, meta.CountryOfOrigin = typ, mango.CountryOfType(typ)
	}
	meta.DateStart = scraper.YearDate(page.Find(sel.Release).First().Text())
	meta.ProviderPublisher = strings.Join(scraper.Texts(page.Find(sel.Publisher).First().Find("a")), ", ")
}

// keiyoushi's status words (MadaraBase.kt), checked in this order
var statuses = []struct {
	status metadata.Status
	words  []string
}{
	{metadata.StatusFinished, []string{"completed", "completo", "completado", "concluído", "concluido", "finalizado", "achevé", "terminé", "hoàn thành", "مكتملة", "مكتمل", "已完结", "tamamlandı", "đã hoàn thành", "завершено", "tamamlanan", "complété"}},
	{metadata.StatusReleasing, []string{"ongoing", "on going", "updating", "продолжается", "em lançamento", "em andamento", "en cours", "ativo", "lançando", "đang tiến hành", "còn nữa", "devam ediyor", "in corso", "in arrivo", "مستمرة", "مستمر", "en curso", "emision", "curso", "en marcha", "publicandose", "publicándose", "en emision", "连载中", "đang làm", "em postagem", "devam eden", "em progresso", "atualizações semanais", "виходить", "đang ra"}},
	{metadata.StatusHiatus, []string{"on hold", "hiatus", "pausado", "en espera", "durduruldu", "beklemede", "đang chờ", "متوقف", "en pause", "заморожено", "en attente", "tạm dừng"}},
	{metadata.StatusCancelled, []string{"canceled", "cancelled", "cancelado", "iptal edildi", "đã hủy", "ملغي", "abandonné", "заброшено", "annulé"}},
}

// Status: The metadata status of a madara status text, "UNKNOWN" if it's none of keiyoushi's.
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
