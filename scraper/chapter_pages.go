package scraper

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

func (s *Scraper) ChapterPages(ctx context.Context, store mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	var pages []mangadata.Page

	pagesURL := chapter.URL
	if s.config.PagesURL != nil {
		pagesURL = s.config.PagesURL(chapter)
	}
	err := s.scrape(rod.WithAction(ctx, s.config.PageExtractor.Action), pagesURL, chapter.URL, func(p *page) (err error) {
		pages, err = s.parsePages(p, chapter)
		return err
	})
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages found: %w", mango.ErrNotFound)
	}

	return pages, nil
}

// parsePages: The pages of the chapter in its page.
func (s *Scraper) parsePages(p *page, chapter mango.Chapter) ([]mangadata.Page, error) {
	elements := p.doc.Find(s.config.PageExtractor.Selector)
	var urls []string
	if s.config.PageExtractor.URLs != nil {
		urls = s.config.PageExtractor.URLs(elements)
	} else {
		elements.Each(func(_ int, selection *goquery.Selection) {
			urls = append(urls, s.config.PageExtractor.URL(selection))
		})
	}

	pages := make([]mangadata.Page, 0, len(urls))
	for _, u := range urls {
		// e.g. a page break without its image
		if strings.TrimSpace(u) == "" {
			continue
		}
		// as is if it can't be resolved
		if abs := p.absoluteURL(u); abs != "" {
			u = abs
		}
		page, err := s.urlToPage(chapter, u)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func (s *Scraper) urlToPage(chapter mango.Chapter, url string) (*mango.Page, error) {
	ext, err := mango.ImageExtension(url)
	if err != nil {
		return nil, err
	}

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
	}, nil
}
