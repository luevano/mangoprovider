package scraper

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

func (s *Scraper) ChapterPages(ctx context.Context, store mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	var pages []mangadata.Page

	// data shared with the collector callbacks
	data := colly.NewContext()
	data.Put("chapter", chapter)
	data.Put("pages", &pages)

	collector := s.getPagesCollector()
	if err := scrape(ctx, collector, chapter.URL, data); err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages found: %w", mango.ErrNotFound)
	}

	return pages, nil
}

// Get the pages collector, the actual scraping logic is defined here.
func (s *Scraper) getPagesCollector() *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, s.config, rod.ActionPage)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		defer recoverScrape(e)
		elements := e.DOM.Find(s.config.PageExtractor.Selector)
		chapter := e.Request.Ctx.GetAny("chapter").(mango.Chapter)
		pages := e.Request.Ctx.GetAny("pages").(*[]mangadata.Page)

		if s.config.PageExtractor.URLs != nil {
			urls := s.config.PageExtractor.URLs(elements)
			for _, u := range urls {
				*pages = append(*pages, s.urlToPage(chapter, u))
			}
			return
		}

		elements.Each(func(_ int, selection *goquery.Selection) {
			url := s.config.PageExtractor.URL(selection)
			*pages = append(*pages, s.urlToPage(chapter, url))
		})
	})
	return collector
}

func (s *Scraper) urlToPage(chapter mango.Chapter, url string) *mango.Page {
	ext := filepath.Ext(url)
	// remove some query params from the extension
	ext = strings.Split(ext, "?")[0]

	headers := map[string]string{
		"Referer":    chapter.URL,
		"Accept":     "image/webp,image/apng,image/*,*/*;q=0.8",
		"User-Agent": s.options.UserAgent,
	}
	return &mango.Page{
		Ext:      ext,
		URL:      url,
		Headers:  headers,
		Chapter_: &chapter,
	}
}
