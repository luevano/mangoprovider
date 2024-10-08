package scraper

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

func (s *Scraper) VolumeChapters(_ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter

	// need an identifiable string for the cache
	cacheID := fmt.Sprintf("%s-chapters", volume.Manga_.URL)

	found, err := store.GetChapters(cacheID, volume, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log("found chapters in cache for manga %q with id %q", volume.Manga_.String(), volume.Manga_.ID)
		return chapters, nil
	}

	ctx := colly.NewContext()
	ctx.Put("volume", volume)
	ctx.Put("chapters", &chapters)

	// TODO: check if using this URL is good enough, only works for sources that
	// don't provide volumes and thus everything is in the manga url
	collector := s.getChaptersCollector()
	err = collector.Request(http.MethodGet, volume.Manga_.URL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	collector.Wait()

	if s.config.ReverseChapters {
		slices.Reverse(chapters)
	}

	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].Info().Number < chapters[j].Info().Number
	})

	// TODO: only cache if there are chapters (len > 0)?
	err = store.SetChapters(cacheID, chapters)
	if err != nil {
		return nil, err
	}

	return chapters, nil
}

func (s *Scraper) getChaptersCollector() *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, s.config, rod.ActionChapter)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.config.ChapterExtractor.Selector)
		volume := e.Request.Ctx.GetAny("volume").(mango.Volume)
		chapters := e.Request.Ctx.GetAny("chapters").(*[]mangadata.Chapter)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.config.ChapterExtractor.URL(selection)
			url := e.Request.AbsoluteURL(link)
			title := strings.TrimSpace(mango.CleanString(s.config.ChapterExtractor.Title(selection)))

			// default temp number
			chapterNumber := float32(e.Index)

			// check for matched groups, to extract the chapter number safely
			matchGroups := mango.ReNamedGroups(mango.ChapterNameRegex, title)
			match := strings.TrimSpace(matchGroups[mango.ChapterNumberID])

			// if the groups matching fail, it means it probably only contains the chapter number
			if match == "" {
				match = mango.ChapterNumberRegex.FindString(title)
			}
			if match != "" {
				match = strings.Replace(match, "-", ".", 1)
				number, err := strconv.ParseFloat(match, 32)
				if err == nil {
					chapterNumber = float32(number)
				}
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
