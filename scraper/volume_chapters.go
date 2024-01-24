package scraper

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (s *Scraper) VolumeChapters(_ctx context.Context, store gokv.Store, volume mango.Volume) ([]libmangal.Chapter, error) {
	var chapters []libmangal.Chapter

	// need an identifiable string for the cache
	cacheID := fmt.Sprintf("%s-chapters", volume.Manga_.URL)

	found, err := store.Get(cacheID, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("found chapters in cache for manga %q with id %q", volume.Manga_.Title, volume.Manga_.ID))
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

	// TODO: only cache if there are chapters (len > 0)?
	err = store.Set(cacheID, chapters)
	if err != nil {
		return nil, err
	}

	return chapters, nil
}

func (s *Scraper) getChaptersCollector() *colly.Collector {
	collector := s.collector.Clone()
	setCollectorOnRequest(collector, s.config, "chapter")
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.config.ChapterExtractor.Selector)
		volume := e.Request.Ctx.GetAny("volume").(mango.Volume)
		chapters := e.Request.Ctx.GetAny("chapters").(*[]libmangal.Chapter)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.config.ChapterExtractor.URL(selection)
			url := e.Request.AbsoluteURL(link)
			title := cleanString(s.config.ChapterExtractor.Title(selection))

			match := chapterNumberRegex.FindString(title)
			chapterNumber := float32(e.Index)
			if match != "" {
				number, err := strconv.ParseFloat(match, 32)
				if err == nil {
					chapterNumber = float32(number)
				}
			}

			var chapterDate libmangal.Date
			if s.config.ChapterExtractor.Date != nil {
				chapterDate = s.config.ChapterExtractor.Date(selection)
			}

			var scanlationGroup string
			if s.config.ChapterExtractor.ScanlationGroup != nil {
				scanlationGroup = s.config.ChapterExtractor.ScanlationGroup(selection)
			}

			c := mango.Chapter{
				Title:           title,
				ID:              s.config.ChapterExtractor.ID(url),
				URL:             url,
				Number:          chapterNumber,
				Date:            chapterDate,
				ScanlationGroup: scanlationGroup,
				Volume_:         &volume,
			}
			*chapters = append(*chapters, c)
		})
	})
	return collector
}
