package scraper

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

func (s *Scraper) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter

	found, err := store.GetChapters(chaptersKey(volume.Manga_), volume, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		s.log.Log("found chapters in cache for manga %q with id %q", volume.Manga_.String(), volume.Manga_.ID)
		s.startFromChapters(volume.Manga_, chapters)
		return chapters, nil
	}

	request, err := s.chaptersRequest(ctx, *volume.Manga_)
	if err != nil {
		return nil, err
	}
	err = s.scrapeRequest(rod.WithAction(ctx, s.config.ChapterExtractor.Action), request, volume.Manga_.URL, func(p *page) error {
		chapters = s.parseChapters(p, volume)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := s.setChapters(store, volume.Manga_, chapters); err != nil {
		return nil, err
	}
	return chapters, nil
}

// chaptersKey: The cache key of the manga's chapters (needs an identifiable string).
func chaptersKey(manga *mango.Manga) string {
	return fmt.Sprintf("%s-chapters", manga.URL)
}

// setChapters: Puts the chapters, as parsed from the page, in order and caches them.
func (s *Scraper) setChapters(store mango.Store, manga *mango.Manga, chapters []mangadata.Chapter) error {
	if s.config.ReverseChapters {
		slices.Reverse(chapters)
	}
	numberExtraChapters(chapters)

	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].Info().Number < chapters[j].Info().Number
	})
	s.startFromChapters(manga, chapters)

	// could be a markup change on the site, try again next time
	if len(chapters) == 0 {
		return nil
	}
	return store.SetChapters(chaptersKey(manga), chapters)
}

// chaptersOnMangaPage: The chapters are listed on the manga page.
func (s *Scraper) chaptersOnMangaPage() bool {
	return s.config.ChaptersRequest == nil && s.config.ChaptersURL == nil
}

// chaptersRequest: The request listing the manga's chapters, its page by default.
func (s *Scraper) chaptersRequest(ctx context.Context, manga mango.Manga) (Request, error) {
	switch {
	case s.config.ChaptersRequest != nil:
		return s.config.ChaptersRequest(manga, func() (*goquery.Selection, error) {
			var doc *goquery.Selection
			err := s.scrape(ctx, manga.URL, manga.URL, func(p *page) error {
				doc = p.doc.Selection
				return nil
			})
			return doc, err
		})
	case s.config.ChaptersURL != nil:
		return Request{URL: s.config.ChaptersURL(manga)}, nil
	}
	return Request{URL: manga.URL}, nil
}

// parseChapters: The chapters in the page, in its order.
func (s *Scraper) parseChapters(p *page, volume mango.Volume) []mangadata.Chapter {
	var chapters []mangadata.Chapter
	p.doc.Find(s.config.ChapterExtractor.Selector).Each(func(_ int, selection *goquery.Selection) {
		url := p.absoluteURL(s.config.ChapterExtractor.URL(selection))
		title := strings.TrimSpace(mango.CleanString(s.config.ChapterExtractor.Title(selection)))
		id := s.config.ChapterExtractor.ID(url)
		if id == "" {
			s.log.Log("skipping chapter %q: no id in %q", title, url)
			return
		}

		parseNumber := mango.ParseChapterNumber
		if s.config.ChapterExtractor.Number != nil {
			parseNumber = s.config.ChapterExtractor.Number
		}
		chapterNumber, ok := parseNumber(title)
		if !ok {
			// numbered once all chapters are in order, see numberExtraChapters
			chapterNumber = -1
		}

		var chapterDate metadata.Date
		if s.config.ChapterExtractor.Date != nil {
			chapterDate = s.config.ChapterExtractor.Date(selection)
		}

		var scanlationGroup string
		if s.config.ChapterExtractor.ScanlationGroup != nil {
			scanlationGroup = s.config.ChapterExtractor.ScanlationGroup(selection)
		}

		c := mango.Chapter{
			Title:           mango.ParseChapterTitle(title),
			ID:              id,
			URL:             url,
			Number:          chapterNumber,
			Date:            chapterDate,
			ScanlationGroup: scanlationGroup,
			Volume_:         &volume,
		}
		chapters = append(chapters, &c)
	})
	return chapters
}

// numberExtraChapters: Chapters without a number (e.g. "Extra") go right after the previous one,
// on a number no other chapter has.
func numberExtraChapters(chapters []mangadata.Chapter) {
	taken := map[float32]bool{}
	for _, chapter := range chapters {
		if n := chapter.Info().Number; n >= 0 {
			taken[n] = true
		}
	}
	var last float32
	for _, chapter := range chapters {
		c := chapter.(*mango.Chapter)
		if c.Number < 0 {
			c.Number = mango.NextFreeExtraNumber(last, taken)
			taken[c.Number] = true
		}
		last = c.Number
	}
}
