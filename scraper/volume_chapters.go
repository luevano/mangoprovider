package scraper

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
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

	// data shared with the collector callbacks
	data := colly.NewContext()
	data.Put("volume", volume)
	data.Put("chapters", &chapters)

	chaptersURL := volume.Manga_.URL
	if s.config.ChaptersURL != nil {
		chaptersURL = s.config.ChaptersURL(*volume.Manga_)
	}
	collector := s.getChaptersCollector()
	if err := scrape(ctx, collector, chaptersURL, data); err != nil {
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

func (s *Scraper) getChaptersCollector() *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, requestChapter)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		defer recoverScrape(e)
		elements := e.DOM.Find(s.config.ChapterExtractor.Selector)
		volume := e.Request.Ctx.GetAny("volume").(mango.Volume)
		chapters := e.Request.Ctx.GetAny("chapters").(*[]mangadata.Chapter)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.config.ChapterExtractor.URL(selection)
			url := e.Request.AbsoluteURL(link)
			title := strings.TrimSpace(mango.CleanString(s.config.ChapterExtractor.Title(selection)))

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
				ID:              s.config.ChapterExtractor.ID(url),
				URL:             url,
				Number:          chapterNumber,
				Date:            chapterDate,
				ScanlationGroup: scanlationGroup,
				Volume_:         &volume,
			}
			*chapters = append(*chapters, &c)
		})
	})
	return collector
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
