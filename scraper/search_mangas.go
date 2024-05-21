package scraper

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
	"github.com/philippgille/gokv"
)

func (s *Scraper) SearchMangas(_ctx context.Context, store gokv.Store, query string) ([]libmangal.Manga, error) {
	var mangas []libmangal.Manga

	matchGroups := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)
	mangaID, byID := matchGroups[mango.MangaQueryIDName]
	mangaID = strings.Trim(mangaID, "/")

	var err error
	var cacheID string
	var searchURL string
	if byID {
		if s.config.GenerateSearchByIDURL == nil {
			return nil, fmt.Errorf("Can't search by ID, %q doesn't implement GenerateSearchByIDURL", s.config.Name)
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

	found, err := store.Get(cacheID, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("found mangas in cache with query %q", query))
		return mangas, nil
	}

	ctx := colly.NewContext()
	ctx.Put("mangas", &mangas)

	var collector *colly.Collector
	if byID {
		if s.config.MangaByIDExtractor == nil {
			return nil, fmt.Errorf("Can't search by ID, %q doesn't implement MangaByIDExtractor", s.config.Name)
		}
		collector = s.getMangaCollector(mangaID)
	} else {
		collector = s.getMangasCollector()
	}

	err = collector.Request("GET", searchURL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	collector.Wait()

	err = store.Set(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}

// Get the manga collector, the actual scraping logic is defined here.
func (s *Scraper) getMangaCollector(id string) *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, s.config, rod.ActionManga)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		selection := e.DOM.Find(s.config.MangaByIDExtractor.Selector).First()
		mangas := e.Request.Ctx.GetAny("mangas").(*[]libmangal.Manga)

		title := mango.CleanString(s.config.MangaByIDExtractor.Title(selection))
		m := mango.Manga{
			Title:         title,
			AnilistSearch: title,
			URL:           e.Request.URL.String(),
			ID:            id,
			Cover:         s.config.MangaByIDExtractor.Cover(selection),
		}
		*mangas = append(*mangas, &m)
	})
	return collector
}

// Get the mangas collector, the actual scraping logic is defined here.
func (s *Scraper) getMangasCollector() *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, s.config, rod.ActionManga)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.config.MangaExtractor.Selector)
		mangas := e.Request.Ctx.GetAny("mangas").(*[]libmangal.Manga)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.config.MangaExtractor.URL(selection)
			url := e.Request.AbsoluteURL(link)
			title := mango.CleanString(s.config.MangaExtractor.Title(selection))
			m := mango.Manga{
				Title:         title,
				AnilistSearch: title,
				URL:           url,
				ID:            s.config.MangaExtractor.ID(url),
				Cover:         s.config.MangaExtractor.Cover(selection),
			}
			*mangas = append(*mangas, &m)
		})
	})
	return collector
}
