package scraper

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

func (s *Scraper) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	if id, byID := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)[mango.MangaQueryID]; byID {
		return mango.SearchByID(s.MangaByID(ctx, store, id))
	}

	var mangas []mangadata.Manga
	searchURL, err := s.config.GenerateSearchURL(s.config.BaseURL, query)
	if err != nil {
		return nil, err
	}
	found, err := store.GetMangas(searchURL, query, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		return mangas, nil
	}

	// data shared with the collector callbacks
	data := colly.NewContext()
	data.Put("mangas", &mangas)
	if err := scrape(ctx, s.getMangasCollector(), searchURL, data); err != nil {
		return nil, err
	}

	// could be a markup change on the site, try again next time
	if len(mangas) == 0 {
		return mangas, nil
	}
	return mangas, store.SetMangas(searchURL, mangas)
}

// MangaByID scrapes the manga's page, found false if the site answers 404.
func (s *Scraper) MangaByID(ctx context.Context, store mango.Store, id string) (mangadata.Manga, bool, error) {
	id = strings.Trim(id, "/")
	if s.config.GenerateSearchByIDURL == nil {
		return nil, false, fmt.Errorf("search by id: %w (no GenerateSearchByIDURL)", mango.ErrNotSupported)
	}
	if s.config.MangaByIDExtractor == nil {
		return nil, false, fmt.Errorf("search by id: %w (no MangaByIDExtractor)", mango.ErrNotSupported)
	}
	mangaURL, err := s.config.GenerateSearchByIDURL(s.config.BaseURL, mango.EscapePath(id))
	if err != nil {
		return nil, false, err
	}

	cacheID := "mid:" + id
	var mangas []mangadata.Manga
	if found, err := store.GetMangas(cacheID, cacheID, &mangas); err != nil {
		return nil, false, err
	} else if found && len(mangas) == 1 {
		return mangas[0], true, nil
	}

	mangas = nil
	data := colly.NewContext()
	data.Put("mangas", &mangas)
	if err := scrape(ctx, s.getMangaCollector(id), mangaURL, data); err != nil {
		if errors.Is(err, mango.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if data.GetAny(notFoundKey) != nil {
		return nil, false, nil
	}
	if len(mangas) == 0 {
		return nil, false, fmt.Errorf("GET %s: no manga in the page", mangaURL)
	}
	return mangas[0], true, store.SetMangas(cacheID, mangas[:1])
}

const notFoundKey = "notFound"

var errorPagePath = regexp.MustCompile(`^/(400|404)/?$`)

// TODO: define metadata scraping logic
// Get the manga collector, the actual scraping logic is defined here.
func (s *Scraper) getMangaCollector(id string) *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, requestManga)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		defer recoverScrape(e)
		// some sites (weebcentral) redirect unknown mangas to an error page that answers 200
		if errorPagePath.MatchString(e.Request.URL.Path) {
			e.Request.Ctx.Put(notFoundKey, true)
			return
		}
		selection := e.DOM.Find(s.config.MangaByIDExtractor.Selector).First()
		if selection.Length() == 0 {
			return
		}
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
	s.setCollectorOnRequest(collector, requestManga)
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
