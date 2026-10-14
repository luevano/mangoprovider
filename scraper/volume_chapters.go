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

	// need an identifiable string for the cache
	cacheID := fmt.Sprintf("%s-chapters", volume.Manga_.URL)

	found, err := store.GetChapters(cacheID, volume, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		s.log.Log("found chapters in cache for manga %q with id %q", volume.Manga_.String(), volume.Manga_.ID)
		return chapters, nil
	}

	chaptersURL := volume.Manga_.URL
	if s.config.ChaptersURL != nil {
		chaptersURL = s.config.ChaptersURL(*volume.Manga_)
	}
	err = s.scrape(rod.WithAction(ctx, s.config.ChapterExtractor.Action), chaptersURL, volume.Manga_.URL, func(p *page) error {
		chapters = s.parseChapters(p, volume)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.config.ReverseChapters {
		slices.Reverse(chapters)
	}
	numberExtraChapters(chapters)

	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].Info().Number < chapters[j].Info().Number
	})

	// could be a markup change on the site, try again next time
	if len(chapters) == 0 {
		return chapters, nil
	}
	err = store.SetChapters(cacheID, chapters)
	if err != nil {
		return nil, err
	}

	return chapters, nil
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

// numberExtraChapters: Chapters without a number (e.g. "Extra") go right after the previous one.
func numberExtraChapters(chapters []mangadata.Chapter) {
	var last float32
	for _, chapter := range chapters {
		c := chapter.(*mango.Chapter)
		if c.Number < 0 {
			c.Number = mango.NextExtraNumber(last)
		}
		last = c.Number
	}
}
