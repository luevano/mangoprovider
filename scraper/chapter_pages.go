package scraper

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (s *Scraper) ChapterPages(_ctx context.Context, store gokv.Store, chapter mango.Chapter) ([]libmangal.Page, error) {
	var pages []libmangal.Page

	// need an identifiable string for the cache
	cacheID := fmt.Sprintf("%s-pages", chapter.URL)

	found, err := store.Get(cacheID, &pages)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("found pages in cache for manga %q with id %q", chapter.Volume_.Manga_.Title, chapter.Volume_.Manga_.ID))
		return pages, nil
	}

	ctx := colly.NewContext()
	ctx.Put("chapter", chapter)
	ctx.Put("pages", &pages)

	collector := s.getPagesCollector()
	err = collector.Request(http.MethodGet, chapter.URL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	collector.Wait()

	err = store.Set(cacheID, pages)
	if err != nil {
		return nil, err
	}

	return pages, nil
}

// Get the pages collector, the actual scraping logic is defined here.
func (s *Scraper) getPagesCollector() *colly.Collector {
	collector := s.collector.Clone()
	setCollectorOnRequest(collector, s.config, "page")
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.config.PageExtractor.Selector)
		chapter := e.Request.Ctx.GetAny("chapter").(mango.Chapter)
		pages := e.Request.Ctx.GetAny("pages").(*[]libmangal.Page)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.config.PageExtractor.URL(selection)
			ext := filepath.Ext(link)
			// remove some query params from the extension
			ext = strings.Split(ext, "?")[0]

			headers := map[string]string{
				"Referer":    chapter.URL,
				"Accept":     "image/webp,image/apng,image/*,*/*;q=0.8",
				"User-Agent": mango.UserAgent,
			}

			p := mango.Page{
				Extension: ext,
				URL:       link,
				Headers:   headers,
				Chapter_:  &chapter,
			}
			*pages = append(*pages, p)
		})
	})
	return collector
}
