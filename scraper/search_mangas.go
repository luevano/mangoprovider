package scraper

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

func (s *Scraper) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	var mangas []mangadata.Manga

	matchGroups := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)
	mangaID, byID := matchGroups[mango.MangaQueryID]
	mangaID = strings.Trim(mangaID, "/")

	var err error
	var cacheID string
	var searchURL string
	if byID {
		if s.config.GenerateSearchByIDURL == nil {
			return nil, fmt.Errorf("search by id: %w (no GenerateSearchByIDURL)", mango.ErrNotSupported)
		}
		searchURL, err = s.config.GenerateSearchByIDURL(s.config.BaseURL, mangaID)
		cacheID = fmt.Sprintf("mid:%s", mangaID)
	} else {
		searchURL, err = s.config.GenerateSearchURL(s.config.BaseURL, query)
		cacheID = searchURL
	}
	if err != nil {
		return nil, err
	}

	found, err := store.GetMangas(cacheID, query, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		return mangas, nil
	}

	// data shared with the collector callbacks
	data := colly.NewContext()
	data.Put("mangas", &mangas)

	var collector *colly.Collector
	if byID {
		if s.config.MangaByIDExtractor == nil {
			return nil, fmt.Errorf("search by id: %w (no MangaByIDExtractor)", mango.ErrNotSupported)
		}
		collector = s.getMangaCollector(mangaID)
	} else {
		collector = s.getMangasCollector()
	}

	if err := scrape(ctx, collector, searchURL, data); err != nil {
		return nil, err
	}

	// could be a markup change on the site, try again next time
	if len(mangas) == 0 {
		return mangas, nil
	}
	err = store.SetMangas(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}

// TODO: define metadata scraping logic
// Get the manga collector, the actual scraping logic is defined here.
func (s *Scraper) getMangaCollector(id string) *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, rod.ActionManga)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		defer recoverScrape(e)
		selection := e.DOM.Find(s.config.MangaByIDExtractor.Selector).First()
		mangas := e.Request.Ctx.GetAny("mangas").(*[]mangadata.Manga)

		var meta metadata.Metadata = &mangadata.Metadata{}
		m := mango.Manga{
			Title:     mango.CleanString(s.config.MangaByIDExtractor.Title(selection)),
			URL:       e.Request.URL.String(),
			ID:        id,
			Cover:     s.config.MangaByIDExtractor.Cover(selection),
			Metadata_: &meta,
		}
		*mangas = append(*mangas, &m)
	})
	return collector
}

// TODO: define metadata scraping logic
// Get the mangas collector, the actual scraping logic is defined here.
func (s *Scraper) getMangasCollector() *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, rod.ActionManga)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		defer recoverScrape(e)
		elements := e.DOM.Find(s.config.MangaExtractor.Selector)
		mangas := e.Request.Ctx.GetAny("mangas").(*[]mangadata.Manga)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.config.MangaExtractor.URL(selection)
			url := e.Request.AbsoluteURL(link)
			var meta metadata.Metadata = &mangadata.Metadata{}
			m := mango.Manga{
				Title:     mango.CleanString(s.config.MangaExtractor.Title(selection)),
				URL:       url,
				ID:        s.config.MangaExtractor.ID(url),
				Cover:     s.config.MangaExtractor.Cover(selection),
				Metadata_: &meta,
			}
			*mangas = append(*mangas, &m)
		})
	})
	return collector
}
