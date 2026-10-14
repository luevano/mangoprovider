package scraper

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

// MetadataExtractor: The manga's metadata from its page. Search results don't have it (it'd be
// a request each): it's read where the page is fetched anyway, MangaByID and MangaVolumes, and
// cached, so later searches show it too.
type MetadataExtractor struct {
	// IDCode: The metadata's provider id code (e.g. "mp"), see metadata.ID.
	IDCode string
	// Extract: Fills meta from the manga's page: alternate titles, start date, status,
	// authors, artists, genres, tags, summary, cover, ... The scraper fills in the rest: the
	// title (the manga's if none), the cover (same), the url and the provider ids.
	Extract func(page *goquery.Selection, meta *mangadata.Metadata)
	// StartFromChapters: Without a start date in the page, the oldest chapter's date is the
	// start (e.g. madara sites show none): close enough for matching (a year off only costs a
	// tie-break) and ComicInfo.
	StartFromChapters bool
}

// metadataKey: The cache key of a manga's metadata.
func metadataKey(manga *mango.Manga) string {
	return "meta:" + manga.ID
}

// needsMetadata: The manga has no metadata yet, or only an empty one (search results); one set
// by the caller (e.g. AniList's) is left as is.
func needsMetadata(manga *mango.Manga) bool {
	if manga.Metadata_ == nil || *manga.Metadata_ == nil {
		return true
	}
	meta, ok := (*manga.Metadata_).(*mangadata.Metadata)
	return ok && meta.Title() == ""
}

// setMetadata: Sets the manga's metadata, shared with its other copies (the search result).
func setMetadata(manga *mango.Manga, meta *mangadata.Metadata) {
	if manga.Metadata_ == nil {
		manga.Metadata_ = new(metadata.Metadata)
	}
	*manga.Metadata_ = meta
}

// ownMetadata: The manga's metadata if it's the one this scraper extracted.
func (s *Scraper) ownMetadata(manga *mango.Manga) (*mangadata.Metadata, bool) {
	if s.config.MetadataExtractor == nil || manga.Metadata_ == nil {
		return nil, false
	}
	meta, ok := (*manga.Metadata_).(*mangadata.Metadata)
	return meta, ok && meta.Title() != "" && meta.ProviderIDCode == s.config.MetadataExtractor.IDCode
}

// extractMetadata: The manga's metadata in its page.
func (s *Scraper) extractMetadata(p *page, manga *mango.Manga) *mangadata.Metadata {
	meta := &mangadata.Metadata{}
	s.config.MetadataExtractor.Extract(p.doc.Selection, meta)
	if meta.Title() == "" {
		meta.EnglishTitle = manga.Title
	}
	title := meta.Title()
	var synonyms []string
	for _, t := range meta.Synonyms {
		if t = mango.CleanString(t); t != "" && t != title && !slices.Contains(synonyms, t) {
			synonyms = append(synonyms, t)
		}
	}
	meta.Synonyms = synonyms
	meta.Summary = strings.TrimSpace(meta.Summary)
	if meta.CoverImage == "" {
		meta.CoverImage = manga.Cover
	} else if abs := p.absoluteURL(meta.CoverImage); abs != "" {
		meta.CoverImage = abs
	}
	meta.SourceURL = p.url.String()
	meta.ProviderID = manga.ID
	meta.ProviderIDCode = s.config.MetadataExtractor.IDCode
	return meta
}

// cachedMetadata: Sets the manga's metadata from the cache, if it needs it and it's there.
func (s *Scraper) cachedMetadata(store mango.Store, manga *mango.Manga) bool {
	if s.config.MetadataExtractor == nil || !needsMetadata(manga) {
		return true
	}
	var cached []mangadata.Manga
	key := metadataKey(manga)
	found, err := store.GetMangas(key, key, &cached)
	if err != nil || !found || len(cached) != 1 {
		return false
	}
	meta, ok := cached[0].Metadata().(*mangadata.Metadata)
	if !ok || meta.Title() == "" {
		return false
	}
	setMetadata(manga, meta)
	return true
}

// cacheMetadata: Caches the manga's metadata (with the manga, the store keeps mangas).
func (s *Scraper) cacheMetadata(store mango.Store, manga *mango.Manga, meta *mangadata.Metadata) error {
	var m metadata.Metadata = meta
	cached := *manga
	cached.Metadata_ = &m
	return store.SetMangas(metadataKey(manga), []mangadata.Manga{&cached})
}

// startFromChapters: Without a start date, the oldest of the chapters' dates, see
// MetadataExtractor.StartFromChapters.
func (s *Scraper) startFromChapters(manga *mango.Manga, chapters []mangadata.Chapter) {
	meta, ok := s.ownMetadata(manga)
	if !ok || !s.config.MetadataExtractor.StartFromChapters || meta.DateStart != (metadata.Date{}) {
		return
	}
	for _, chapter := range chapters {
		date := chapter.(*mango.Chapter).Date
		if date != (metadata.Date{}) && (meta.DateStart == (metadata.Date{}) || before(date, meta.DateStart)) {
			meta.DateStart = date
		}
	}
}

func before(a, b metadata.Date) bool {
	if a.Year != b.Year {
		return a.Year < b.Year
	}
	if a.Month != b.Month {
		return a.Month < b.Month
	}
	return a.Day < b.Day
}

// Texts: The cleaned up text of each element, without empty ones or repeats.
func Texts(s *goquery.Selection) []string {
	var texts []string
	s.Each(func(_ int, e *goquery.Selection) {
		if t := mango.CleanString(e.Text()); t != "" && !slices.Contains(texts, t) {
			texts = append(texts, t)
		}
	})
	return texts
}

var yearRegex = regexp.MustCompile(`\b(1[89]\d\d|2\d\d\d)\b`)

// YearDate: The year in text (e.g. "Released: 1997") as a date, none if there's none.
func YearDate(text string) metadata.Date {
	m := yearRegex.FindString(text)
	if m == "" {
		return metadata.Date{}
	}
	year, _ := strconv.Atoi(m)
	return metadata.Date{Year: year}
}
