package scraper

import (
	"context"
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
	"github.com/philippgille/gokv"
)

func (s *Scraper) SearchMangas(_ctx context.Context, store gokv.Store, query string) ([]libmangal.Manga, error) {
	var mangas []libmangal.Manga

	searchURL, err := s.config.GenerateSearchURL(s.config.BaseURL, query)
	if err != nil {
		return nil, err
	}

	found, err := store.Get(searchURL, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("found mangas in cache with query %q", query))
		return mangas, nil
	}

	ctx := colly.NewContext()
	ctx.Put("mangas", &mangas)

	collector := s.getMangasCollector()
	err = collector.Request("GET", searchURL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	collector.Wait()

	err = store.Set(searchURL, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}

// Get the mangas collector, the actual scraping logic is defined here.
func (s *Scraper) getMangasCollector() *colly.Collector {
	collector := s.collector.Clone()
	setCollectorOnRequest(collector, s.config, rod.ActionManga)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.config.MangaExtractor.Selector)
		mangas := e.Request.Ctx.GetAny("mangas").(*[]libmangal.Manga)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.config.MangaExtractor.URL(selection)
			url := e.Request.AbsoluteURL(link)
			title := mango.CleanString(s.config.MangaExtractor.Title(selection))
			if title != "" {
				m := mango.Manga{
					Title:         title,
					AnilistSearch: title,
					URL:           url,
					ID:            s.config.MangaExtractor.ID(url),
					Cover:         s.config.MangaExtractor.Cover(selection),
				}
				*mangas = append(*mangas, &m)
			} else {
				mango.Log(fmt.Sprintf("Warning, omitting manga with empty title (%s)", url))
			}
		})
	})
	return collector
}
